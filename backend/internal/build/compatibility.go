package build

import (
	"sort"
	"strings"
)

// CheckCompatibility compares only selected CPU/motherboard sockets and
// selected RAM models/motherboard RAM types. Missing components are permitted
// for draft evaluation, but missing metadata must never silently match.
func CheckCompatibility(components []Component, models []RAMModel) []Issue {
	issues := []Issue{}
	var cpus, boards []Component
	for _, component := range components {
		switch component.Product.Category {
		case "cpu":
			cpus = append(cpus, component)
			if strings.TrimSpace(component.Specs.CPUSocket) == "" {
				issues = append(issues, Issue{Code: "MISSING_CPU_SOCKET", LeftID: component.Product.ID})
			}
		case "motherboard":
			boards = append(boards, component)
			if strings.TrimSpace(component.Specs.CPUSocket) == "" {
				issues = append(issues, Issue{Code: "MISSING_CPU_SOCKET", LeftID: component.Product.ID})
			}
			if strings.TrimSpace(component.Specs.RAMType) == "" {
				issues = append(issues, Issue{Code: "MISSING_RAM_TYPE", LeftID: component.Product.ID})
			}
		}
	}
	for _, model := range models {
		if strings.TrimSpace(model.RAMType) == "" {
			issues = append(issues, Issue{Code: "MISSING_RAM_TYPE", LeftID: model.ID})
		}
	}
	for _, board := range boards {
		for _, cpu := range cpus {
			if cpu.Specs.CPUSocket != "" && board.Specs.CPUSocket != "" && cpu.Specs.CPUSocket != board.Specs.CPUSocket {
				issues = append(issues, Issue{Code: "CPU_SOCKET_MISMATCH", LeftID: cpu.Product.ID, RightID: board.Product.ID})
			}
		}
		for _, model := range models {
			if model.RAMType != "" && board.Specs.RAMType != "" && model.RAMType != board.Specs.RAMType {
				issues = append(issues, Issue{Code: "RAM_TYPE_MISMATCH", LeftID: model.ID, RightID: board.Product.ID})
			}
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.LeftID != b.LeftID {
			return a.LeftID < b.LeftID
		}
		return a.RightID < b.RightID
	})
	return issues
}
