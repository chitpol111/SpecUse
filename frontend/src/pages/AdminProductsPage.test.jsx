import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import App from "../App";

const user = { email: "admin@example.test" };
const profile = { display_name: "Admin", role: "admin" };
const session = { access_token: "access", refresh_token: "refresh" };
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
      element.dispatchEvent(new Event("input", { bubbles: true }));
    });
  }

  async function signInAsAdminAndOpenProducts() {
    global.fetch.mockImplementation(async (url, options = {}) => {
      if (url.endsWith("/auth/login")) return response(200, { data: { session } });
      if (url.endsWith("/me")) return response(200, { data: { user, profile } });
      if (url.includes("/admin/products") && options.method === "POST") {
        const product = { id: "cpu-1", ...JSON.parse(options.body) };
        products = [product];
        return response(201, { data: { product } });
      }
      if (url.endsWith("/admin/products/cpu-1") && options.method === "PATCH") {
        const changes = JSON.parse(options.body);
        products = products.map((product) => ({ ...product, ...changes }));
        return response(200, { data: { product: products[0] } });
      }
      if (url.endsWith("/admin/products/cpu-1") && options.method === "DELETE") {
        products = products.map((product) => ({ ...product, is_active: false }));
        return response(204);
      }
      if (url.includes("/admin/products")) {
        return response(200, {
          data: { items: products, page: 1, page_size: 20, total: products.length },
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

  it("creates and soft-deactivates a product without allowing category changes", async () => {
    await signInAsAdminAndOpenProducts();
    const ids = Array.from(container.querySelectorAll("[id]"), (element) => element.id);
    expect(new Set(ids).size).toBe(ids.length);

    setValue("#product-name", "Test CPU");
    setValue("#product-brand", "Example");
    setValue("#product-price", "1000");
    setValue("#product-specs", JSON.stringify({ cpu_socket: "AM5" }));
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
    expect(JSON.parse(updateCall[1].body)).toEqual({ price_thb: 1200, is_active: true, name: "Test CPU", brand: "Example", specs: { cpu_socket: "AM5" } });

    window.confirm = jest.fn(() => true);
    const deactivateButton = Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent === "ปิดใช้งาน");
    await act(async () => deactivateButton.click());
    expect(window.confirm).toHaveBeenCalledTimes(1);
    expect(container.querySelector("main").textContent).toContain("ปิดใช้งาน Test CPU แล้ว");
    expect(container.querySelector("main").textContent).toContain("ปิดใช้งาน");
  });
});
