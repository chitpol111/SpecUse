import { createAuthClient, readStoredSession } from "./authApi";

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
    global.fetch = vi.fn();
  });
  afterEach(() => {
    delete global.fetch;
  });

  it("restores a stored session, saves refreshed tokens, and removes them on logout", async () => {
    const first = createAuthClient(vi.fn());
    global.fetch.mockResolvedValueOnce(response(200, { data: { session } }));
    await first.login({});
    expect(readStoredSession()).toEqual({ session, warning: "" });

    const onChange = vi.fn();
    const restored = createAuthClient(onChange, { initialSession: readStoredSession().session });
    global.fetch
      .mockResolvedValueOnce(response(401, { error: { code: "UNAUTHORIZED", message: "expired" } }))
      .mockResolvedValueOnce(response(200, { data: { session: nextSession } }))
      .mockResolvedValueOnce(response(200, { data: { user, profile } }))
      .mockResolvedValueOnce(response(204));
    await restored.restoreProfile();
    expect(readStoredSession()).toEqual({ session: nextSession, warning: "" });
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user, profile });
    await restored.logout();
    expect(readStoredSession()).toEqual({ session: null, warning: "" });
  });

  it("clears the stored session when its refresh token is rejected", async () => {
    const first = createAuthClient(vi.fn());
    global.fetch.mockResolvedValueOnce(response(200, { data: { session } }));
    await first.login({});
    const restored = createAuthClient(vi.fn(), { initialSession: readStoredSession().session });
    global.fetch
      .mockResolvedValueOnce(response(401, { error: { code: "UNAUTHORIZED", message: "expired" } }))
      .mockResolvedValueOnce(response(401, { error: { code: "INVALID_REFRESH_TOKEN", message: "expired" } }));
    await expect(restored.restoreProfile()).rejects.toThrow("expired");
    expect(readStoredSession().session).toBeNull();
  });

  it("keeps the stored session when restoring the profile fails due to a network error", async () => {
    const first = createAuthClient(vi.fn());
    global.fetch.mockResolvedValueOnce(response(200, { data: { session } }));
    await first.login({});
    const restored = createAuthClient(vi.fn(), { initialSession: readStoredSession().session });
    global.fetch.mockRejectedValueOnce(new Error("offline"));
    await expect(restored.restoreProfile()).rejects.toThrow("เชื่อมต่อ Backend ไม่ได้");
    expect(readStoredSession().session).toEqual(session);
  });

  it("warns when browser storage is unavailable without losing the active session", async () => {
    const warning = vi.fn();
    const onChange = vi.fn();
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("blocked"); });
    try {
      const client = createAuthClient(onChange, { onStorageStatus: warning });
      global.fetch.mockResolvedValueOnce(response(200, { data: { session } }));
      await client.login({});
      expect(warning).toHaveBeenCalledWith(expect.stringContaining("เก็บสถานะเข้าสู่ระบบไม่ได้"));
      expect(onChange).toHaveBeenLastCalledWith({ authenticated: true, user: null, profile: null });
      expect(readStoredSession().session).toBeNull();
    } finally {
      setItem.mockRestore();
    }
  });

  it("reports when logout cannot remove stored credentials", async () => {
    const warning = vi.fn();
    const onChange = vi.fn();
    const client = createAuthClient(onChange, { onStorageStatus: warning });
    global.fetch.mockResolvedValueOnce(response(200, { data: { session } })).mockResolvedValueOnce(response(204));
    await client.login({});
    const removeItem = vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => { throw new Error("blocked"); });
    try {
      await expect(client.logout()).rejects.toMatchObject({ code: "SESSION_STORAGE_CLEAR_FAILED" });
      expect(warning).toHaveBeenLastCalledWith(expect.stringContaining("ล้างข้อมูลเข้าสู่ระบบที่เก็บไว้ไม่ได้"));
      expect(onChange).toHaveBeenLastCalledWith({ authenticated: false, user: null, profile: null });
    } finally {
      removeItem.mockRestore();
    }
  });

  it("reports malformed stored sessions without using them", () => {
    window.sessionStorage.setItem("specuse.auth.session", "{invalid");
    expect(readStoredSession()).toEqual({
      session: null,
      warning: expect.stringContaining("ข้อมูลเข้าสู่ระบบที่เก็บไว้ไม่ถูกต้อง"),
    });
  });

  it("uses protected private-draft endpoints and patches only metadata", async () => {
    const client = createAuthClient(vi.fn());
    const build = {
      id: "build", name: "Draft", description: "", visibility: "private", created_at: "now", updated_at: "now",
      items: [{ product_id: "cpu", quantity: 2, owned_quantity: 1, product_summary: { name: "CPU", category: "cpu", is_active: false } }],
      ram_items: [{ ram_model_id: "ram", required_modules: 4, owned_modules: 2, ram_model_summary: { ram_type: "DDR4" } }],
      availability: { ready_for_validation: false, issues: [{ message: "inactive" }] },
    };
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(201, { data: { build } }))
      .mockResolvedValueOnce(response(200, { data: { items: [build], page: 1, page_size: 20, total: 1 } }))
      .mockResolvedValueOnce(response(200, { data: { build } }))
      .mockResolvedValueOnce(response(204));
    await client.login({});
    const body = { name: "Draft", items: [], ram_items: [] };
    await expect(client.createBuild(body)).resolves.toEqual(build);
    await expect(client.listBuilds()).resolves.toMatchObject({ total: 1 });
    await expect(client.updateBuildDetails("build", "Draft", "")).resolves.toEqual(build);
    await client.deleteBuild("build");
    expect(global.fetch.mock.calls[1][1]).toMatchObject({ method: "POST", body: JSON.stringify(body), headers: { "Content-Type": "application/json", Authorization: "Bearer first" } });
    expect(global.fetch.mock.calls[3][1].body).toBe(JSON.stringify({ name: "Draft", description: "" }));
    expect(global.fetch.mock.calls[4][1].method).toBe("DELETE");
  });

  it("rejects malformed draft responses", async () => {
    const client = createAuthClient(vi.fn());
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { build: { items: [] } } }));
    await client.login({});
    await expect(client.getBuild("build")).rejects.toThrow("ไม่ตรงกับ API contract");
  });

  it("refreshes once for simultaneous 401s and retries with the new token", async () => {
    const onChange = vi.fn();
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
    const onChange = vi.fn();
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
    const onChange = vi.fn();
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
    const onChange = vi.fn();
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
    const onChange = vi.fn();
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
    const onChange = vi.fn();
    const client = createAuthClient(onChange);
    global.fetch.mockResolvedValue(response(201, { data: { session: null, email_confirmation_required: false } }));
    await expect(client.register({})).rejects.toThrow("session");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("updates the display name and uses the new session after changing the password", async () => {
    const onChange = vi.fn();
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
    const onChange = vi.fn();
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
    const onChange = vi.fn();
    const client = createAuthClient(onChange);
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { user, session: null } }));
    await client.login({ email: user.email, password: "old-password" });
    await expect(client.changePassword("old-password", "new-password")).rejects.toThrow("รหัสผ่านอาจเปลี่ยนแล้ว");
    expect(onChange).toHaveBeenLastCalledWith({ authenticated: false, user: null, profile: null });
  });

  it("uses the admin catalog filters and validates product mutation responses", async () => {
    const client = createAuthClient(vi.fn());
    const product = {
      id: "cpu-1",
      name: "Example CPU",
      category: "cpu",
      brand: "Example",
      price_thb: 5000,
      is_active: true,
      revision: "a".repeat(64),
    };
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, {
        data: { items: [product], page: 2, page_size: 10, total: 11, revision_format: "sha256" },
      }))
      .mockResolvedValueOnce(response(201, { data: { product } }));

    await client.login({ email: user.email, password: "password" });
    await expect(client.fetchAdminProducts({
      category: "cpu",
      search: "Example CPU",
      page: 2,
      pageSize: 10,
    })).resolves.toEqual({ items: [product], page: 2, page_size: 10, total: 11, revision_format: "sha256" });
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
    const client = createAuthClient(vi.fn());
    global.fetch.mockResolvedValueOnce(response(200, { data: { session } }));
    await client.login({ email: user.email, password: "password" });

    await expect(client.updateAdminProduct("cpu-1", { category: "gpu" }))
      .rejects.toThrow("ไม่สามารถเปลี่ยนหมวดหมู่สินค้าได้");
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it("rejects an admin catalog response without a revision", async () => {
    const client = createAuthClient(vi.fn());
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: {
        items: [{ id: "cpu-1", name: "CPU", category: "cpu", brand: "Example",
          price_thb: 1000, is_active: true }],
        page: 1, page_size: 20, total: 1, revision_format: "sha256",
      } }));
    await client.login({ email: user.email, password: "password" });
    await expect(client.fetchAdminProducts()).rejects.toThrow("รูปแบบไม่ถูกต้อง");
  });

  it("rejects an empty admin catalog from an older backend", async () => {
    const client = createAuthClient(vi.fn());
    global.fetch
      .mockResolvedValueOnce(response(200, { data: { session } }))
      .mockResolvedValueOnce(response(200, { data: { items: [], page: 1, page_size: 20, total: 0 } }));
    await client.login({ email: user.email, password: "password" });
    await expect(client.fetchAdminProducts()).rejects.toThrow("รูปแบบไม่ถูกต้อง");
  });
});
