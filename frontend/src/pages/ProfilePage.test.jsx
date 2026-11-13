import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import App from "../App";

const user = { email: "student@example.test" };
const profile = { display_name: "Student", role: "member" };
const session = { access_token: "access", refresh_token: "refresh" };
const response = (status, data) => ({
  ok: status >= 200 && status < 300,
  status,
  json: async () => data,
});

describe("profile page", () => {
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
    act(() => root.render(
      <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <App />
      </MemoryRouter>,
    ));
  }

  async function signInAndOpenProfile() {
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { user, profile } }));
    render("/login");
    container.querySelector('input[name="email"]').value = user.email;
    container.querySelector('input[name="password"]').value = "old-password";
    await act(async () => container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    await act(async () => container.querySelector('main a[href="/profile"]').dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 })));
  }

  it("asks guests to sign in without requesting private data", () => {
    render("/profile");
    expect(container.querySelector("main").textContent).toContain("เข้าสู่ระบบเพื่อดูข้อมูลส่วนตัว");
    expect(container.querySelector('main a[href="/login"]')).not.toBeNull();
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("shows account details and saves the display name", async () => {
    await signInAndOpenProfile();
    expect(container.querySelector("main").textContent).toContain(user.email);
    expect(container.querySelector("main").textContent).toContain("สมาชิก");
    await act(async () => Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "แก้ไขข้อมูลส่วนตัว").click());
    container.querySelector("#profile-name").value = "  New name  ";
    global.fetch.mockResolvedValueOnce(response(200, { data: { profile: { ...profile, display_name: "New name" } } }));
    await act(async () => container.querySelector("#profile-name").form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(container.querySelector("main").textContent).toContain("บันทึกชื่อที่แสดงแล้ว");
    expect(container.querySelector("main").textContent).toContain("New name");
    expect(JSON.parse(global.fetch.mock.calls[2][1].body)).toEqual({ display_name: "New name" });
  });

  it("rejects mismatched passwords without sending a request", async () => {
    await signInAndOpenProfile();
    container.querySelector("#current-password").value = "old-password";
    container.querySelector("#new-password").value = "new-password";
    container.querySelector("#confirm-password").value = "different-password";
    await act(async () => container.querySelector("#current-password").form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(container.querySelector('[role="alert"]').textContent).toContain("ไม่ตรงกัน");
    expect(global.fetch).toHaveBeenCalledTimes(2);
  });

  it("reports password errors and clears fields only after success", async () => {
    await signInAndOpenProfile();
    const form = container.querySelector("#current-password").form;
    container.querySelector("#current-password").value = "old-password";
    container.querySelector("#new-password").value = "new-password";
    container.querySelector("#confirm-password").value = "new-password";
    global.fetch
      .mockResolvedValueOnce(response(401, { error: { code: "INVALID_CREDENTIALS", message: "รหัสผ่านปัจจุบันไม่ถูกต้อง" } }))
      .mockResolvedValueOnce(response(200, { data: { user, session: { access_token: "new-access", refresh_token: "new-refresh" } } }));
    await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(container.querySelector('[role="alert"]').textContent).toContain("รหัสผ่านปัจจุบันไม่ถูกต้อง");
    expect(container.querySelector("#current-password").value).toBe("old-password");
    await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(container.querySelector('[role="status"]').textContent).toContain("เปลี่ยนรหัสผ่านแล้ว");
    expect(container.querySelector("#current-password").value).toBe("");
  });

  it("explains an unusable replacement session even after signing out locally", async () => {
    await signInAndOpenProfile();
    container.querySelector("#current-password").value = "old-password";
    container.querySelector("#new-password").value = "new-password";
    container.querySelector("#confirm-password").value = "new-password";
    global.fetch.mockResolvedValueOnce(response(200, { data: { user, session: null } }));
    await act(async () => container.querySelector("#current-password").form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(container.querySelector('[role="alert"]').textContent).toContain("รหัสผ่านอาจเปลี่ยนแล้ว");
    expect(container.querySelector('main a[href="/login"]')).not.toBeNull();
  });
});
