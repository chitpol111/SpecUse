const API_BASE_URL = (
  process.env.REACT_APP_API_BASE_URL || "http://127.0.0.1:8080/api/v1"
).replace(/\/+$/, "");

function isValidProduct(product) {
  if (
    !product ||
    typeof product.id !== "string" ||
    typeof product.name !== "string" ||
    typeof product.category !== "string" ||
    typeof product.brand !== "string" ||
    !Number.isInteger(product.price_thb) ||
    typeof product.is_active !== "boolean"
  ) {
    return false;
  }

  const specs = product.specs || {};
  if (product.category === "cpu") {
    return typeof specs.cpu_socket === "string" && specs.cpu_socket.length > 0;
  }
  if (product.category === "motherboard") {
    return (
      typeof specs.cpu_socket === "string" &&
      specs.cpu_socket.length > 0 &&
      typeof specs.ram_type === "string" &&
      specs.ram_type.length > 0
    );
  }
  if (product.category === "ram") {
    return (
      typeof product.ram_model_id === "string" &&
      product.ram_model_id.length > 0 &&
      typeof specs.ram_type === "string" &&
      (specs.modules_per_package === 1 || specs.modules_per_package === 2)
    );
  }
  return true;
}

async function request(path, { method = "GET", body, signal } = {}) {
  let response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method,
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    });
  } catch (error) {
    if (error.name === "AbortError") {
      throw error;
    }
    throw new Error(
      "เชื่อมต่อ Backend ไม่ได้ ตรวจสอบว่า Backend กำลังรันอยู่และอนุญาต CORS สำหรับ URL ของ Frontend",
    );
  }

  let payload;
  try {
    payload = await response.json();
  } catch {
    throw new Error(`Backend ส่งข้อมูลกลับมาอ่านไม่ได้ (HTTP ${response.status})`);
  }

  if (!response.ok) {
    throw new Error(
      payload?.error?.message || `Backend ตอบกลับด้วยสถานะ HTTP ${response.status}`,
    );
  }
  if (!payload || !Object.prototype.hasOwnProperty.call(payload, "data")) {
    throw new Error("รูปแบบคำตอบจาก Backend ไม่ตรงกับ API contract");
  }

  return payload.data;
}

export async function fetchProducts({ signal } = {}) {
  const pageSize = 100;
  const firstPage = await request(`/products?page=1&page_size=${pageSize}`, { signal });
  if (
    !firstPage ||
    !Array.isArray(firstPage.items) ||
    !firstPage.items.every(isValidProduct) ||
    !Number.isInteger(firstPage.total) ||
    firstPage.total < 0
  ) {
    throw new Error("รายการสินค้าจาก Backend มีรูปแบบไม่ถูกต้อง");
  }

  const products = [...firstPage.items];
  const pageCount = Math.ceil(firstPage.total / pageSize);
  for (let page = 2; page <= pageCount; page += 1) {
    const nextPage = await request(`/products?page=${page}&page_size=${pageSize}`, { signal });
    if (
      !nextPage ||
      !Array.isArray(nextPage.items) ||
      !nextPage.items.every(isValidProduct)
    ) {
      throw new Error("รายการสินค้าจาก Backend มีรูปแบบไม่ถูกต้อง");
    }
    products.push(...nextPage.items);
  }

  if (products.length !== firstPage.total) {
    throw new Error("จำนวนสินค้าที่ได้รับไม่ตรงกับข้อมูล pagination จาก Backend");
  }

  return products.filter((product) => product.is_active);
}

export function validateBuild(build, { signal } = {}) {
  return request("/builds/validate", { method: "POST", body: build, signal });
}
