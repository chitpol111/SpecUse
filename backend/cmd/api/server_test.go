package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

// runBlockedRequest starts a real HTTP request on loopback and leaves the
// handler blocked so lifecycle tests can exercise shutdown during a request.
func runBlockedRequest(t *testing.T, timeout time.Duration) (context.CancelFunc, func(), <-chan error, <-chan error, <-chan struct{}) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	shutdownStarted := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "finished")
	})}
	server.RegisterOnShutdown(func() { close(shutdownStarted) })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, server, listener, timeout) }()
	transport := &http.Transport{Proxy: nil}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	requestDone := make(chan error, 1)
	go func() {
		response, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			defer response.Body.Close()
			var body []byte
			body, err = io.ReadAll(response.Body)
			if err == nil && (response.StatusCode != 200 || string(body) != "finished") {
				err = errors.New("request did not complete successfully")
			}
		}
		requestDone <- err
	}()
	t.Cleanup(func() {
		unblock()
		cancel()
		_ = server.Close()
		transport.CloseIdleConnections()
	})
	select {
	case <-started:
	case err := <-requestDone:
		t.Fatalf("request never reached handler: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("request did not start")
	}
	return cancel, unblock, done, requestDone, shutdownStarted
}

func TestServeWaitsForActiveRequest(t *testing.T) {
	cancel, unblock, done, requestDone, shutdownStarted := runBlockedRequest(t, 3*time.Second)
	cancel()
	select {
	case <-shutdownStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not start")
	}
	select {
	case err := <-done:
		t.Fatalf("serve returned before active request completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	unblock()
	select {
	case err := <-requestDone:
		if err != nil {
			t.Fatalf("in-flight request was interrupted: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("request did not complete")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("graceful shutdown failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not return after request completion")
	}
}

func TestServeClosesConnectionsOnShutdownDeadline(t *testing.T) {
	cancel, _, done, requestDone, _ := runBlockedRequest(t, 50*time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected shutdown deadline error, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not enforce deadline")
	}
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("expected forced connection closure")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("connection was not closed")
	}
}

func TestServeReturnsListenerErrors(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_ = listener.Close()
	if err := serve(context.Background(), &http.Server{}, listener, time.Second); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("expected listener error, got %v", err)
	}
}
