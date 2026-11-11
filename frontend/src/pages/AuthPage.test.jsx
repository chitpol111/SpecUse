import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import App from "../App";

const session = { access_token: "access", refresh_token: "refresh" };
const profile = { display_name: "Student", role: "member" };
const user = { email: "student@example.test" };
const response = (status, data) => ({
  ok: status >= 200 && status < 300,
  status,
  json: async () => data,
});

describe("account pages", () => {
  let container;
  let root;

  beforeEach(() => {
    global.IS_REACT_ACT_ENVIRONMENT = true;
    global.fetch = jest.fn();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    delete global.IS_REACT_ACT_ENVIRONMENT;
    delete global.fetch;
  });

  function render(path) {
    act(() => {
      root.render(
        <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
          <App />
        </MemoryRouter>,
      );
    });
  }

  async function submit() {
    container.querySelector('input[name="email"]').value = "student@example.test";
    container.querySelector('input[name="password"]').value = "test-password";
    if (container.querySelector('input[name="displayName"]')) {
      container.querySelector('input[name="displayName"]').value = " Student ";
    }
    await act(async () => {
      container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    });
  }

  it.each([
    ["/login", "/auth/login", 200],
    ["/register", "/auth/register", 201],
  ])("connects %s and loads the backend profile", async (path, endpoint, status) => {
    const data = { session, ...(path === "/register" ? { email_confirmation_required: false } : {}) };
    global.fetch
      .mockResolvedValueOnce(response(status, { data }))
      .mockResolvedValueOnce(response(200, { data: { user, profile } }));
    render(path);
    await submit();
    expect(global.fetch).toHaveBeenNthCalledWith(1, expect.stringContaining(endpoint), expect.objectContaining({
      method: "POST",
      body: JSON.stringify(path === "/register"
        ? { email: "student@example.test", password: "test-password", display_name: "Student" }
        : { email: "student@example.test", password: "test-password" }),
    }));
    expect(global.fetch).toHaveBeenNthCalledWith(2, expect.stringContaining("/me"), expect.objectContaining({
      headers: { Authorization: "Bearer access" },
    }));
    expect(container.querySelector('[role="status"]').textContent).toContain("Student (member)");
    expect(container.textContent).toContain("ออกจากระบบ");
  });

  it("reports configuration errors without retrying signup", async () => {
    global.fetch.mockResolvedValue(response(503, {
      error: { code: "AUTH_CONFIGURATION_ERROR", message: "provider error" },
    }));
    render("/register");
    await submit();
    expect(global.fetch).toHaveBeenCalledTimes(1);
    expect(container.querySelector('[role="alert"]').textContent).toContain("ติดต่อผู้ดูแล");
    expect(container.textContent).not.toContain("ออกจากระบบ");
  });

  it("does not repeat registration when profile retrieval fails", async () => {
    global.fetch
      .mockResolvedValueOnce(response(201, { data: { session, email_confirmation_required: false } }))
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce(response(200, { data: { user, profile } }));
    render("/register");
    await submit();
    expect(container.textContent).toContain("เข้าสู่ระบบแล้ว แต่ยังไม่ได้รับข้อมูลสมาชิก");
    expect(container.querySelector('[role="alert"]').textContent).toContain("โหลดข้อมูลสมาชิกไม่สำเร็จ");
    await act(async () => {
      Array.from(container.querySelectorAll("button")).find((button) => button.textContent.includes("ลองโหลด")).click();
    });
    expect(container.querySelector('[role="status"]').textContent).toContain("Student (member)");
    expect(global.fetch.mock.calls.filter(([url]) => url.endsWith("/auth/register"))).toHaveLength(1);
  });

  it("clears the local session and reports a failed logout", async () => {
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { user, profile } }))
      .mockRejectedValueOnce(new Error("offline"));
    render("/login");
    await submit();
    await act(async () => {
      Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "ออกจากระบบ").click();
    });
    expect(container.textContent).not.toContain("Student (member)");
    expect(container.querySelector('[role="alert"]').textContent).toContain("อาจยังไม่เพิกถอน");
  });
});
