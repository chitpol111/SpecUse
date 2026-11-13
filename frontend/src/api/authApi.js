import { API_BASE_URL } from "./specUseApi";

class AuthError extends Error {
  constructor(message, status, code) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request(path, { method = "GET", body, token, expectedStatus = 200 } = {}) {
  let response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers: {
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
  } catch {
    throw new AuthError("เชื่อมต่อ Backend ไม่ได้ กรุณาลองอีกครั้ง", 0);
  }

  if (response.status === 204 && response.ok && expectedStatus === 204) return null;

  let payload;
  try {
    payload = await response.json();
  } catch {
    throw new AuthError(`Backend ส่งข้อมูลกลับมาอ่านไม่ได้ (HTTP ${response.status})`, response.status);
  }
  if (!response.ok) {
    const code = payload?.error?.code;
    const message = code === "AUTH_CONFIGURATION_ERROR"
      ? "ระบบสมัครสมาชิกยังตั้งค่าไม่พร้อม กรุณาติดต่อผู้ดูแล"
      : payload?.error?.message || `Backend ตอบกลับด้วยสถานะ HTTP ${response.status}`;
    throw new AuthError(message, response.status, code);
  }
  if (response.status !== expectedStatus) {
    throw new AuthError(`Backend ตอบกลับไม่ตรงกับ API contract (HTTP ${response.status})`, response.status);
  }
  if (!payload || !Object.prototype.hasOwnProperty.call(payload, "data")) {
    throw new AuthError("รูปแบบคำตอบจาก Backend ไม่ตรงกับ API contract", response.status);
  }
  return payload.data;
}

function validSession(session) {
  return session &&
    typeof session.access_token === "string" && session.access_token.length > 0 &&
    typeof session.refresh_token === "string" && session.refresh_token.length > 0;
}

const SESSION_KEY = "specuse.auth.session";

export function readStoredSession() {
  let stored;
  try {
    stored = window.sessionStorage.getItem(SESSION_KEY);
  } catch {
    return { session: null, warning: "เบราว์เซอร์ไม่อนุญาตให้เก็บสถานะเข้าสู่ระบบ การรีเฟรชอาจทำให้ต้องเข้าสู่ระบบใหม่" };
  }
  if (stored === null) return { session: null, warning: "" };

  let session;
  try {
    session = JSON.parse(stored);
  } catch {
    return { session: null, warning: "ข้อมูลเข้าสู่ระบบที่เก็บไว้ไม่ถูกต้อง กรุณาเข้าสู่ระบบใหม่" };
  }
  if (!validSession(session)) {
    return { session: null, warning: "ข้อมูลเข้าสู่ระบบที่เก็บไว้ไม่ถูกต้อง กรุณาเข้าสู่ระบบใหม่" };
  }
  return { session, warning: "" };
}

function validProfile(profile) {
  return profile && typeof profile.display_name === "string" &&
    ["member", "admin"].includes(profile.role);
}

function validBuildSummary(build) {
  return build && typeof build.id === "string" && typeof build.name === "string" &&
    typeof build.description === "string" && build.visibility === "private" &&
    typeof build.created_at === "string" && typeof build.updated_at === "string";
}

function validBuild(build) {
  return validBuildSummary(build) && Array.isArray(build.items) &&
    build.items.every((item) => item && typeof item.product_id === "string" &&
      Number.isInteger(item.quantity) && Number.isInteger(item.owned_quantity) &&
      typeof item.product_summary?.name === "string" &&
      typeof item.product_summary?.category === "string" &&
      typeof item.product_summary?.is_active === "boolean") &&
    Array.isArray(build.ram_items) &&
    build.ram_items.every((item) => item && typeof item.ram_model_id === "string" &&
      Number.isInteger(item.required_modules) && Number.isInteger(item.owned_modules) &&
      typeof item.ram_model_summary?.ram_type === "string") &&
    typeof build.availability?.ready_for_validation === "boolean" &&
    Array.isArray(build.availability?.issues) &&
    build.availability.issues.every((issue) => issue && typeof issue.message === "string");
}

export function createAuthClient(onChange, { initialSession = null, onStorageStatus = (message) => {
  if (message) console.error(message);
} } = {}) {
  let session = initialSession;
  let user = null;
  let profile = null;
  let generation = 0;
  let refreshPromise = null;

  function notify() {
    onChange({ authenticated: Boolean(session), user, profile });
  }

  function clearSession() {
    generation += 1;
    session = null;
    user = null;
    profile = null;
    let storageError = null;
    try {
      window.sessionStorage.removeItem(SESSION_KEY);
      onStorageStatus("");
    } catch {
      storageError = new AuthError(
        "ล้างข้อมูลเข้าสู่ระบบที่เก็บไว้ไม่ได้ กรุณาปิดแท็บและล้างข้อมูลเว็บไซต์ในเบราว์เซอร์",
        undefined,
        "SESSION_STORAGE_CLEAR_FAILED",
      );
      onStorageStatus(storageError.message);
    }
    notify();
    if (storageError) throw storageError;
  }

  function setSession(next) {
    if (!validSession(next)) {
      throw new AuthError("Backend ไม่ได้ส่ง session ที่ใช้งานได้ กรุณาติดต่อผู้ดูแล", undefined, "INVALID_SESSION");
    }
    session = next;
    try {
      window.sessionStorage.setItem(SESSION_KEY, JSON.stringify({
        access_token: next.access_token,
        refresh_token: next.refresh_token,
      }));
      onStorageStatus("");
    } catch {
      let warning = "เก็บสถานะเข้าสู่ระบบไม่ได้ การรีเฟรชอาจทำให้ต้องเข้าสู่ระบบใหม่";
      try {
        window.sessionStorage.removeItem(SESSION_KEY);
      } catch {
        warning += " และล้างสถานะเก่าไม่ได้ กรุณาปิดแท็บและล้างข้อมูลเว็บไซต์ในเบราว์เซอร์";
      }
      onStorageStatus(warning);
    }
    notify();
  }

  async function authenticate(path, body, isRegister) {
    const started = generation;
    const data = await request(path, { method: "POST", body, expectedStatus: isRegister ? 201 : 200 });
    if (started !== generation) {
      throw new AuthError("สถานะการเข้าสู่ระบบเปลี่ยนไป กรุณาลองอีกครั้ง");
    }
    if (isRegister && data?.email_confirmation_required !== false) {
      throw new AuthError("ผลการสมัครไม่ตรงกับ API contract กรุณาติดต่อผู้ดูแล");
    }
    if (!validSession(data?.session)) {
      throw new AuthError("Backend ไม่ได้ส่ง session ที่ใช้งานได้ กรุณาติดต่อผู้ดูแล");
    }
    generation += 1;
    user = null;
    profile = null;
    setSession(data.session);
  }

  async function refresh() {
    if (!session) throw new AuthError("กรุณาเข้าสู่ระบบอีกครั้ง", 401);
    if (refreshPromise) return refreshPromise;
    const started = generation;
    const refreshToken = session.refresh_token;
    const pending = (async () => {
      try {
        const data = await request("/auth/refresh", {
          method: "POST",
          body: { refresh_token: refreshToken },
        });
        if (generation !== started) throw new AuthError("สถานะการเข้าสู่ระบบเปลี่ยนไป");
        setSession(data?.session);
      } catch (error) {
        if (generation === started && (error.status === 401 || error.code === "INVALID_SESSION")) {
          clearSession();
        }
        throw error;
      }
    })();
    refreshPromise = pending;
    try {
      await pending;
    } finally {
      if (refreshPromise === pending) refreshPromise = null;
    }
  }

  async function protectedRequest(path, options = {}) {
    if (!session) throw new AuthError("กรุณาเข้าสู่ระบบอีกครั้ง", 401);
    const started = generation;
    const token = session.access_token;
    try {
      return await request(path, { ...options, token });
    } catch (error) {
      if (error.status !== 401 || error.code !== "UNAUTHORIZED" || generation !== started) throw error;
      if (session?.access_token === token) await refresh();
      if (!session || generation !== started) throw new AuthError("กรุณาเข้าสู่ระบบอีกครั้ง", 401);
      try {
        return await request(path, { ...options, token: session.access_token });
      } catch (retryError) {
        if (retryError.status === 401 && retryError.code === "UNAUTHORIZED" && generation === started) clearSession();
        throw retryError;
      }
    }
  }

  async function fetchAdminProducts({ category = "", search = "", page = 1, pageSize = 20 } = {}) {
    if (!Number.isInteger(page) || page < 1 || !Number.isInteger(pageSize) || pageSize < 1) {
      throw new AuthError("เลขหน้าและจำนวนสินค้าต่อหน้าต้องเป็นจำนวนเต็มบวก");
    }
    const query = new URLSearchParams({
      page: String(page),
      page_size: String(pageSize),
      ...(category ? { category } : {}),
      ...(search ? { search } : {}),
    });
    const data = await protectedRequest(`/admin/products?${query}`);
    if (!data || !Array.isArray(data.items) ||
        data.revision_format !== "sha256" ||
        !Number.isInteger(data.page) || data.page < 1 ||
        !Number.isInteger(data.page_size) || data.page_size < 1 ||
        !Number.isInteger(data.total) || data.total < 0 ||
        !data.items.every(isAdminProduct)) {
      throw new AuthError("รายการสินค้าจาก Backend มีรูปแบบไม่ถูกต้อง");
    }
    return data;
  }

  function isAdminProduct(product) {
    return product && typeof product.id === "string" &&
      typeof product.name === "string" && typeof product.category === "string" &&
      typeof product.brand === "string" && Number.isInteger(product.price_thb) &&
      typeof product.is_active === "boolean" &&
      typeof product.revision === "string" && /^[0-9a-f]{64}$/.test(product.revision);
  }

  async function saveAdminProduct(path, method, body, expectedStatus) {
    const data = await protectedRequest(path, { method, body, expectedStatus });
    if (!isAdminProduct(data?.product)) {
      throw new AuthError("ข้อมูลสินค้าจาก Backend มีรูปแบบไม่ถูกต้อง");
    }
    return data;
  }

  async function buildRequest(path, options) {
    const data = await protectedRequest(path, options);
    if (!validBuild(data?.build)) {
      throw new AuthError("ข้อมูลสเปกจาก Backend ไม่ตรงกับ API contract");
    }
    return data.build;
  }

  async function loadProfile() {
    const started = generation;
    const data = await protectedRequest("/me");
    if (started !== generation) throw new AuthError("สถานะการเข้าสู่ระบบเปลี่ยนไป");
    if (!validProfile(data?.profile) || typeof data?.user?.email !== "string") {
      throw new AuthError("ข้อมูลโปรไฟล์จาก Backend ไม่ตรงกับ API contract");
    }
    user = data.user;
    profile = data.profile;
    notify();
    return profile;
  }

  return {
    register: (body) => authenticate("/auth/register", body, true),
    login: (body) => authenticate("/auth/login", body, false),
    fetchAdminProducts,
    createAdminProduct: (body) => saveAdminProduct(
      "/admin/products",
      "POST",
      body,
      201,
    ),
    updateAdminProduct: async (id, body) => {
      if (Object.prototype.hasOwnProperty.call(body, "category")) {
        throw new AuthError("ไม่สามารถเปลี่ยนหมวดหมู่สินค้าได้");
      }
      return saveAdminProduct(
        `/admin/products/${encodeURIComponent(id)}`,
        "PATCH",
        body,
        200,
      );
    },
    deactivateAdminProduct: (id) => protectedRequest(
      `/admin/products/${encodeURIComponent(id)}`,
      { method: "DELETE", expectedStatus: 204 },
    ),
    async listBuilds(page = 1) {
      const data = await protectedRequest(`/me/builds?page=${page}&page_size=20`);
      if (!data || !Array.isArray(data.items) || !data.items.every(validBuildSummary) ||
          !Number.isInteger(data.page) || !Number.isInteger(data.page_size) ||
          data.page < 1 || data.page_size < 1 || !Number.isInteger(data.total) || data.total < 0) {
        throw new AuthError("รายการสเปกจาก Backend ไม่ตรงกับ API contract");
      }
      return data;
    },
    getBuild: (id) => buildRequest(`/me/builds/${encodeURIComponent(id)}`),
    createBuild: (body) => buildRequest("/me/builds", { method: "POST", body, expectedStatus: 201 }),
    updateBuildDetails: (id, name, description) => buildRequest(
      `/me/builds/${encodeURIComponent(id)}`,
      { method: "PATCH", body: { name, description } },
    ),
    deleteBuild: (id) => protectedRequest(
      `/me/builds/${encodeURIComponent(id)}`,
      { method: "DELETE", expectedStatus: 204 },
    ),
    loadProfile,
    async restoreProfile() {
      const started = generation;
      try {
        return await loadProfile();
      } catch (error) {
        if (generation !== started &&
            (session || !["INVALID_REFRESH_TOKEN", "INVALID_SESSION", "SESSION_STORAGE_CLEAR_FAILED"].includes(error.code))) {
          return;
        }
        throw error;
      }
    },
    async updateProfile(displayName) {
      const started = generation;
      const data = await protectedRequest("/me", {
        method: "PATCH",
        body: { display_name: displayName },
      });
      if (started !== generation) throw new AuthError("สถานะการเข้าสู่ระบบเปลี่ยนไป");
      if (!validProfile(data?.profile)) {
        throw new AuthError("ข้อมูลโปรไฟล์จาก Backend ไม่ตรงกับ API contract");
      }
      profile = data.profile;
      notify();
      return profile;
    },
    async changePassword(currentPassword, newPassword) {
      const started = generation;
      const data = await protectedRequest("/me/password", {
        method: "PATCH",
        body: { current_password: currentPassword, new_password: newPassword },
      });
      if (started !== generation) throw new AuthError("สถานะการเข้าสู่ระบบเปลี่ยนไป");
      if (!validSession(data?.session)) {
        clearSession();
        throw new AuthError("Backend ตอบกลับโดยไม่มี session ที่ใช้งานได้ รหัสผ่านอาจเปลี่ยนแล้ว กรุณาเข้าสู่ระบบใหม่");
      }
      generation += 1;
      refreshPromise = null;
      setSession(data.session);
    },
    async logout() {
      const started = generation;
      try {
        await protectedRequest("/auth/logout", { method: "POST", expectedStatus: 204 });
      } finally {
        if (started === generation) clearSession();
      }
    },
  };
}
