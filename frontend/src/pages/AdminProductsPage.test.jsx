import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import App from "../App";

const user = { email: "admin@example.test" };
const profile = { display_name: "Admin", role: "admin" };
const session = { access_token: "access", refresh_token: "refresh" };
const revision = "a".repeat(64);
const response = (status, data) => ({
  ok: status >= 200 && status < 300,
  status,
  json: async () => data,
});

describe("admin products page", () => {
  let container;
  let root;
  let products;

  beforeEach(() => {
    global.IS_REACT_ACT_ENVIRONMENT = true;
    global.fetch = jest.fn();
    products = [];
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    delete global.IS_REACT_ACT_ENVIRONMENT;
    delete global.fetch;
    delete window.confirm;
  });

  function render(path) {
    act(() => root.render(
      <MemoryRouter initialEntries={[path]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <App />
      </MemoryRouter>,
    ));
  }

  function setValue(selector, value) {
    const element = container.querySelector(selector);
    const setter = Object.getOwnPropertyDescriptor(element.constructor.prototype, "value").set;
    act(() => {
      setter.call(element, value);
      element.dispatchEvent(new Event(element.tagName === "SELECT" ? "change" : "input", { bubbles: true }));
    });
  }

  async function signInAsAdminAndOpenProducts(backendSupportsRevision = true) {
    global.fetch.mockImplementation(async (url, options = {}) => {
      if (url.endsWith("/auth/login")) return response(200, { data: { session } });
      if (url.endsWith("/me")) return response(200, { data: { user, profile } });
      if (url.includes("/admin/products") && options.method === "POST") {
        const product = { id: "cpu-1", ...JSON.parse(options.body), revision };
        products = [product];
        return response(201, { data: { product } });
      }
      if (url.endsWith("/admin/products/cpu-1") && options.method === "PATCH") {
        const { expected_revision, ...changes } = JSON.parse(options.body);
        if (expected_revision !== products[0].revision) {
          return response(409, { error: { code: "CONFLICT", message: "Conflict" } });
        }
        products = products.map((product) => ({ ...product, ...changes, revision: "b".repeat(64) }));
        return response(200, { data: { product: products[0] } });
      }
      if (url.endsWith("/admin/products/cpu-1") && options.method === "DELETE") {
        products = products.map((product) => ({ ...product, is_active: false }));
        return response(204);
      }
      if (url.includes("/admin/products")) {
        return response(200, {
          data: { items: products, page: 1, page_size: 20, total: products.length,
            ...(backendSupportsRevision ? { revision_format: "sha256" } : {}) },
        });
      }
      throw new Error(`Unexpected API request: ${url}`);
    });

    render("/login");
    container.querySelector('input[name="email"]').value = user.email;
    container.querySelector('input[name="password"]').value = "password";
    await act(async () => container.querySelector("form").dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    await act(async () => container.querySelector('a[href="/admin/products"]').click());
  }

  it("asks guests to sign in without requesting the admin catalog", () => {
    render("/admin/products");
    expect(container.querySelector("main").textContent).toContain("เข้าสู่ระบบด้วยบัญชีผู้ดูแล");
    expect(container.querySelector('main a[href="/login"]')).not.toBeNull();
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it("blocks product creation when the backend cannot protect edits", async () => {
    await signInAsAdminAndOpenProducts(false);
    expect(container.querySelector('[role="alert"]').textContent).toContain("รูปแบบไม่ถูกต้อง");
    const form = container.querySelector("#product-name").form;
    expect(form.querySelector('button[type="submit"]').disabled).toBe(true);
    await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(global.fetch.mock.calls.some(([, options]) => options?.method === "POST" && options.body?.includes('"category"'))).toBe(false);
  });

  it("creates and soft-deactivates a product without allowing category changes", async () => {
    await signInAsAdminAndOpenProducts();
    const ids = Array.from(container.querySelectorAll("[id]"), (element) => element.id);
    expect(new Set(ids).size).toBe(ids.length);

    setValue("#product-name", "Test CPU");
    setValue("#product-brand", "Example");
    setValue("#product-price", "1000");
    setValue("#cpu-socket", "AM5");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));

    expect(container.querySelector('[role="status"]').textContent).toContain("เพิ่มสินค้า Test CPU แล้ว");
    const createCall = global.fetch.mock.calls.find(([url, options]) =>
      url.includes("/admin/products") && options?.method === "POST");
    expect(JSON.parse(createCall[1].body)).toEqual({
      name: "Test CPU",
      category: "cpu",
      brand: "Example",
      price_thb: 1000,
      is_active: true,
      specs: { cpu_socket: "AM5" },
    });
    expect(container.querySelector("#product-category").disabled).toBe(false);

    await act(async () => Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "แก้ไข").click());
    expect(container.querySelector("#product-category").disabled).toBe(true);
    setValue("#product-price", "1200");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    expect(container.querySelector('[role="status"]').textContent).toContain("บันทึกการแก้ไข Test CPU แล้ว");
    const updateCall = global.fetch.mock.calls.find(([, options]) => options?.method === "PATCH");
    expect(JSON.parse(updateCall[1].body)).toEqual({ price_thb: 1200, is_active: true, name: "Test CPU", brand: "Example", specs: { cpu_socket: "AM5" }, expected_revision: revision });
    setValue("#product-price", "1300");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    const updateCalls = global.fetch.mock.calls.filter(([, options]) => options?.method === "PATCH");
    expect(JSON.parse(updateCalls[1][1].body).expected_revision).toBe("b".repeat(64));

    window.confirm = jest.fn(() => true);
    const deactivateButton = Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "ปิดใช้งาน");
    await act(async () => deactivateButton.click());
    expect(window.confirm).toHaveBeenCalledTimes(1);
    expect(container.querySelector("main").textContent).toContain("ปิดใช้งาน Test CPU แล้ว");
    expect(container.querySelector("main").textContent).toContain("ปิดใช้งาน");
  });

  it("preserves the form and explains a stale edit conflict", async () => {
    await signInAsAdminAndOpenProducts();
    products = [{ id: "cpu-1", name: "Original", category: "cpu", brand: "Example",
      price_thb: 1000, is_active: true, specs: { cpu_socket: "AM5" }, revision }];
    setValue("#product-search", "Original");
    await act(async () => container.querySelector("#product-search").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    await act(async () => Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "แก้ไข").click());
    setValue("#product-name", "My unsaved edit");
    products = [{ ...products[0], name: "Another admin", revision: "b".repeat(64) }];
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    expect(container.querySelector('[role="alert"]').textContent).toContain("ข้อมูลสินค้าขัดแย้ง");
    expect(container.querySelector("#product-name").value).toBe("My unsaved edit");
    expect(products[0].name).toBe("Another admin");
  });

  it("uses category-specific fields and keeps motherboard advanced metadata when editing", async () => {
    await signInAsAdminAndOpenProducts();
    setValue("#product-name", "Test board");
    setValue("#product-brand", "Example");
    setValue("#product-price", "2000");
    setValue("#product-category", "motherboard");
    setValue("#cpu-socket", "AM5");
    setValue("#ram-type", "DDR5");
    setValue("#product-specs", JSON.stringify({ connectivity_profiles: null, sources: null, evidence: null }));
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    const createCall = global.fetch.mock.calls.find(([url, options]) =>
      url.includes("/admin/products") && options?.method === "POST");
    expect(JSON.parse(createCall[1].body).specs).toEqual({
      cpu_socket: "AM5", ram_type: "DDR5", connectivity_profiles: null, sources: null, evidence: null,
    });
    await act(async () => Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "แก้ไข").click());
    setValue("#product-price", "2200");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    const updateCall = global.fetch.mock.calls.find(([, options]) => options?.method === "PATCH");
    expect(JSON.parse(updateCall[1].body).specs).toEqual({
      cpu_socket: "AM5", ram_type: "DDR5", connectivity_profiles: null, sources: null, evidence: null,
    });
  });

  it("builds optional GPU display metadata from simple fields without inventing other data", async () => {
    await signInAsAdminAndOpenProducts();
    setValue("#product-name", "Test GPU");
    setValue("#product-brand", "Example");
    setValue("#product-price", "5000");
    setValue("#product-category", "gpu");
    setValue("#gpu-chipset", "Test chip");
    setValue("#gpu-vram", "1.28");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    const createCall = global.fetch.mock.calls.find(([url, options]) =>
      url.includes("/admin/products") && options?.method === "POST");
    expect(JSON.parse(createCall[1].body).specs).toEqual({
      display: { chipset_name: "Test chip", vram_capacity_bytes: 1280000000 },
    });
  });

  it("uses the chosen RAM type, package size and existing model ID", async () => {
    await signInAsAdminAndOpenProducts();
    setValue("#product-name", "Test RAM");
    setValue("#product-brand", "Example");
    setValue("#product-price", "1000");
    setValue("#product-category", "ram");
    setValue("#ram-type", "DDR5");
    setValue("#ram-modules", "2");
    setValue("#ram-model-id", "model-id");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    const createCall = global.fetch.mock.calls.find(([url, options]) =>
      url.includes("/admin/products") && options?.method === "POST");
    expect(JSON.parse(createCall[1].body)).toEqual(expect.objectContaining({
      category: "ram",
      ram_model_id: "model-id",
      specs: { ram_type: "DDR5", modules_per_package: 2 },
    }));
  });

  it("preserves storage metadata while updating its displayed capacity", async () => {
    await signInAsAdminAndOpenProducts();
    setValue("#product-name", "Test SSD");
    setValue("#product-brand", "Example");
    setValue("#product-price", "1000");
    setValue("#product-category", "storage");
    setValue("#storage-type", "ssd");
    setValue("#storage-capacity", "500");
    setValue("#product-specs", JSON.stringify({ host: { connection_type: "sata" } }));
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    await act(async () => Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "แก้ไข").click());
    setValue("#storage-capacity", "1000");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    const updateCall = global.fetch.mock.calls.find(([, options]) => options?.method === "PATCH");
    expect(JSON.parse(updateCall[1].body).specs).toEqual({
      display: { media_type: "ssd", capacity_bytes: 1000000000000 },
      host: { connection_type: "sata" },
    });
  });

  it("rejects malformed advanced specs before sending a request", async () => {
    await signInAsAdminAndOpenProducts();
    setValue("#product-name", "Test CPU");
    setValue("#product-brand", "Example");
    setValue("#product-price", "1000");
    setValue("#cpu-socket", "AM5");
    setValue("#product-specs", "{");
    await act(async () => container.querySelector("#product-name").form.dispatchEvent(
      new Event("submit", { bubbles: true, cancelable: true }),
    ));
    expect(container.querySelector('[role="alert"]').textContent).toContain("JSON ที่ถูกต้อง");
    expect(global.fetch.mock.calls.some(([url, options]) =>
      url.includes("/admin/products") && options?.method === "POST")).toBe(false);
  });
});
