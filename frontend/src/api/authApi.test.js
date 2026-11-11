import { createAuthClient } from "./authApi";

const session = { access_token: "first", refresh_token: "old-refresh" };
const nextSession = { access_token: "second", refresh_token: "new-refresh" };
const profile = { display_name: "Admin", role: "admin" };
const user = { email: "admin@example.test" };
const response = (status, payload) => ({
  ok: status >= 200 && status < 300,
  status,
  json: async () => payload,
});

describe("auth API client", () => {
  beforeEach(() => {
    global.fetch = jest.fn();
  });
  afterEach(() => {
    delete global.fetch;
  });

  it("refreshes once for simultaneous 401s and retries with the new token", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    let releaseRefresh;
    global.fetch.mockImplementation(async (url, options) => {
      if (url.endsWith("/auth/login")) return response(200, { data: { session } });
      if (url.endsWith("/auth/refresh")) {
        expect(JSON.parse(options.body)).toEqual({ refresh_token: "old-refresh" });
        await new Promise((resolve) => { releaseRefresh = resolve; });
        return response(200, { data: { session: nextSession } });
      }
      if (options.headers.Authorization === "Bearer first") {
        return response(401, { error: { code: "UNAUTHORIZED", message: "expired" } });
      }
      return response(200, { data: { user, profile } });
    });
    await client.login({ email: "a@example.test", password: "password" });
    const first = client.loadProfile();
    const second = client.loadProfile();
    await new Promise((resolve) => setTimeout(resolve, 0));
    releaseRefresh();
    await expect(Promise.all([first, second])).resolves.toEqual([profile, profile]);
    expect(global.fetch.mock.calls.filter(([url]) => url.endsWith("/auth/refresh"))).toHaveLength(1);
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user, profile });
  });

  it("clears the session when refresh is rejected", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(401, { error: { code: "UNAUTHORIZED", message: "expired" } }))
      .mockResolvedValueOnce(response(401, { error: { code: "INVALID_REFRESH_TOKEN", message: "expired" } }));
    await client.login({ email: "a@example.test", password: "password" });
    await expect(client.loadProfile()).rejects.toThrow("expired");
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: false, user: null, profile: null });
  });

  it("does not refresh on a 403 and preserves the session", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(403, { error: { code: "FORBIDDEN", message: "forbidden" } }));
    await client.login({ email: "a@example.test", password: "password" });
    await expect(client.loadProfile()).rejects.toThrow("forbidden");
    expect(global.fetch).toHaveBeenCalledTimes(2);
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user: null, profile: null });
  });

  it("preserves the session if refresh fails due to a network error", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(401, { error: { code: "UNAUTHORIZED", message: "expired" } }))
      .mockRejectedValueOnce(new Error("offline"));
    await client.login({ email: "a@example.test", password: "password" });
    await expect(client.loadProfile()).rejects.toThrow("เชื่อมต่อ Backend ไม่ได้");
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user: null, profile: null });
  });

  it("does not restore a session from a refresh completed after logout", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    let releaseRefresh;
    global.fetch.mockImplementation(async (url) => {
      if (url.endsWith("/auth/login")) return response(200, { data: { session } });
      if (url.endsWith("/auth/logout")) return response(204);
      if (url.endsWith("/auth/refresh")) {
        await new Promise((resolve) => { releaseRefresh = resolve; });
        return response(200, { data: { session: nextSession } });
      }
      return response(401, { error: { code: "UNAUTHORIZED", message: "expired" } });
    });
    await client.login({ email: "a@example.test", password: "password" });
    const profileRequest = client.loadProfile();
    await new Promise((resolve) => setTimeout(resolve, 0));
    await client.logout();
    releaseRefresh();
    await expect(profileRequest).rejects.toThrow("สถานะการเข้าสู่ระบบเปลี่ยนไป");
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: false, user: null, profile: null });
  });

  it("rejects a signup response without the required session", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    global.fetch.mockResolvedValue(response(201, { data: { session: null, email_confirmation_required: false } }));
    await expect(client.register({})).rejects.toThrow("session");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("updates the display name and uses the new session after changing the password", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    const updatedProfile = { ...profile, display_name: "New name" };
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { user, profile } }))
      .mockResolvedValueOnce(response(200, { data: { profile: updatedProfile } }))
      .mockResolvedValueOnce(response(200, { data: { user, session: nextSession } }))
      .mockResolvedValueOnce(response(200, { data: { user, profile: updatedProfile } }));

    await client.login({ email: user.email, password: "old-password" });
    await client.loadProfile();
    await client.updateProfile("New name");
    await client.changePassword("old-password", "new-password");
    await client.loadProfile();

    expect(global.fetch.mock.calls[2][1]).toEqual(expect.objectContaining({
      method: "PATCH",
      body: JSON.stringify({ display_name: "New name" }),
    }));
    expect(global.fetch.mock.calls[3][1].body).toBe(JSON.stringify({
      current_password: "old-password",
      new_password: "new-password",
    }));
    expect(global.fetch.mock.calls[4][1].headers.Authorization).toBe("Bearer second");
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user, profile: updatedProfile });
  });

  it("does not refresh or clear the session when the current password is wrong", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(401, { error: { code: "INVALID_CREDENTIALS", message: "รหัสผ่านปัจจุบันไม่ถูกต้อง" } }));
    await client.login({ email: user.email, password: "old-password" });
    await expect(client.changePassword("wrong-password", "new-password")).rejects.toThrow("รหัสผ่านปัจจุบันไม่ถูกต้อง");
    expect(global.fetch).toHaveBeenCalledTimes(2);
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user: null, profile: null });
  });

  it("clears the session if a successful password update returns no usable session", async () => {
    const onChange = jest.fn();
    const client = createAuthClient(onChange);
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { user, session: null } }));
    await client.login({ email: user.email, password: "old-password" });
    await expect(client.changePassword("old-password", "new-password")).rejects.toThrow("รหัสผ่านอาจเปลี่ยนแล้ว");
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: false, user: null, profile: null });
  });

  it("uses the admin catalog filters and validates product mutation responses", async () => {
    const client = createAuthClient(jest.fn());
    const product = {
      id: "cpu-1",
      name: "Example CPU",
      category: "cpu",
      brand: "Example",
      price_thb: 5000,
      is_active: true,
    };
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, {
        data: { items: [product], page: 2, page_size: 10, total: 11 },
      }))
      .mockResolvedValueOnce(response(201, { data: { product } }));

    await client.login({ email: user.email, password: "password" });
    await expect(client.fetchAdminProducts({
      category: "cpu",
      search: "Example CPU",
      page: 2,
      pageSize: 10,
    })).resolves.toEqual({ items: [product], page: 2, page_size: 10, total: 11 });
    await expect(client.createAdminProduct({
      name: product.name,
      category: product.category,
      brand: product.brand,
      price_thb: product.price_thb,
      is_active: true,
    })).resolves.toEqual({ product });

    expect(global.fetch.mock.calls[1][0]).toContain("category=cpu");
    expect(global.fetch.mock.calls[1][0]).toContain("search=Example");
    expect(JSON.parse(global.fetch.mock.calls[2][1].body)).toEqual({
      name: product.name,
      category: product.category,
      brand: product.brand,
      price_thb: product.price_thb,
      is_active: true,
    });
  });

  it("does not send an immutable category in an admin product update", async () => {
    const client = createAuthClient(jest.fn());
    global.fetch.mockResolvedValueOnce(response(200, { data: { session } }));
    await client.login({ email: user.email, password: "password" });

    await expect(client.updateAdminProduct("cpu-1", { category: "gpu" }))
      .rejects.toThrow("ไม่สามารถเปลี่ยนหมวดหมู่สินค้าได้");
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });
});
