import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import BuildPage from "./BuildPage";
import { useAuth } from "../auth/AuthContext";
import { fetchProducts, validateBuild } from "../api/specUseApi";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
vi.mock("../api/specUseApi", () => ({ fetchProducts: vi.fn(), validateBuild: vi.fn() }));
vi.mock("../components/SiteHeader", () => ({ default: () => null }));
vi.mock("../components/SiteFooter", () => ({ default: () => null }));

const products = [
  { id: "cpu", name: "CPU A", category: "cpu", brand: "A", price_thb: 1000, specs: { cpu_socket: "AM4" } },
  { id: "board", name: "Board A", category: "motherboard", brand: "A", price_thb: 2000, specs: { cpu_socket: "AM4", ram_type: "DDR4" } },
  { id: "ram", ram_model_id: "ram-model", name: "RAM A", category: "ram", brand: "A", price_thb: 500, specs: { ram_type: "DDR4", modules_per_package: 1 } },
];

describe("saving a build", () => {
  let container;
  let root;
  let client;

  beforeEach(() => {
    global.IS_REACT_ACT_ENVIRONMENT = true;
    window.scrollTo = vi.fn();
    fetchProducts.mockResolvedValue(products);
    validateBuild.mockResolvedValue({
      compatible: true, issues: [], items: [], ram_selections: [], total_price_thb: 3000,
      compatibility: { mode: "gpu_storage_v1", status: "unknown", checks: [] },
    });
    client = { createBuild: vi.fn().mockResolvedValue({}) };
    useAuth.mockReturnValue({ authenticated: true, client });
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.resetAllMocks();
    delete global.IS_REACT_ACT_ENVIRONMENT;
  });

  async function choose(category, name) {
    const nav = container.querySelector('nav[aria-label="หมวดอุปกรณ์"]');
    await act(async () => Array.from(nav.querySelectorAll("button")).find((button) => button.textContent.includes(category)).click());
    const article = Array.from(container.querySelectorAll("article")).find((item) => item.querySelector("h3")?.textContent === name);
    await act(async () => article.querySelector("button").click());
  }

  async function setInput(input, value) {
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  }

  it("saves selected items as a private draft and rejects unsafe RAM counts", async () => {
    await act(async () => root.render(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><BuildPage /></MemoryRouter>));
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    await choose("หน่วยความจำ", "DDR4 RAM");
    await act(async () => Array.from(container.querySelectorAll("button")).find((button) => button.textContent.includes("สร้างชุดสเปก ·")).click());
    const name = Array.from(container.querySelectorAll("label")).find((label) => label.textContent.includes("ชื่อสเปก")).querySelector("input");
    await setInput(name, "  ทดสอบ  ");
    const form = name.closest("form");
    await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(client.createBuild).toHaveBeenCalledWith({
      name: "ทดสอบ", description: "", visibility: "private",
      items: [{ product_id: "cpu", quantity: 1, owned_quantity: 0 }, { product_id: "board", quantity: 1, owned_quantity: 0 }],
      ram_items: [{ ram_model_id: "ram-model", required_modules: 2, owned_modules: 0 }],
    });
    expect(container.textContent).toContain("บันทึกสเปกส่วนตัวแล้ว");
    const ramCount = Array.from(container.querySelectorAll("label")).find((label) => label.textContent.includes("โมดูล RAM ที่ต้องการ")).querySelector("input");
    const previousCalls = validateBuild.mock.calls.length;
    await setInput(ramCount, "9007199254740992");
    expect(validateBuild).toHaveBeenCalledTimes(previousCalls);
    await act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(client.createBuild).toHaveBeenCalledTimes(1);
    expect(container.textContent).toContain("จำนวน RAM ต้องเป็นจำนวนเต็มในช่วงที่รองรับ");
    expect(container.textContent).not.toContain("บันทึกสเปกส่วนตัวแล้ว");
  });
});
