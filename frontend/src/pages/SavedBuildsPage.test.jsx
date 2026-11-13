import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import SavedBuildsPage from "./SavedBuildsPage";
import { useAuth } from "../auth/AuthContext";
import { validateBuild } from "../api/specUseApi";

vi.mock("../auth/AuthContext", () => ({ useAuth: vi.fn() }));
vi.mock("../api/specUseApi", () => ({ validateBuild: vi.fn() }));
vi.mock("../components/SiteHeader", () => ({ default: () => null }));
vi.mock("../components/SiteFooter", () => ({ default: () => null }));

const summary = { id: "one", name: "สเปกแรก", description: "", visibility: "private", created_at: "now", updated_at: "now" };
const draft = {
  ...summary,
  items: [
    { product_id: "one", quantity: 2, owned_quantity: 1, product_summary: { name: "CPU A", category: "cpu", is_active: false } },
    { product_id: "two", quantity: 1, owned_quantity: 0, product_summary: { name: "CPU B", category: "cpu", is_active: true } },
  ],
  ram_items: [
    { ram_model_id: "ram1", required_modules: 2, owned_modules: 1, ram_model_summary: { ram_type: "DDR4" } },
    { ram_model_id: "ram2", required_modules: 4, owned_modules: 4, ram_model_summary: { ram_type: "DDR5" } },
  ],
  availability: { ready_for_validation: true, issues: [] },
};

describe("saved builds", () => {
  let container;
  let root;
  let client;

  beforeEach(() => {
    global.IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    client = {
      listBuilds: vi.fn().mockResolvedValue({ items: [summary], page: 1, page_size: 20, total: 21 }),
      getBuild: vi.fn().mockResolvedValue(draft),
      updateBuildDetails: vi.fn().mockResolvedValue({ ...draft, name: "ชื่อใหม่" }),
      deleteBuild: vi.fn().mockResolvedValue(null),
    };
    useAuth.mockReturnValue({ authenticated: true, client });
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.resetAllMocks();
    delete global.IS_REACT_ACT_ENVIRONMENT;
  });

  async function render() {
    await act(async () => root.render(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><SavedBuildsPage /></MemoryRouter>));
  }
  async function click(text) {
    await act(async () => Array.from(container.querySelectorAll("button")).find((button) => button.textContent.includes(text)).click());
  }

  it("requires login before loading private builds", async () => {
    useAuth.mockReturnValue({ authenticated: false, client });
    await render();
    expect(container.textContent).toContain("เข้าสู่ระบบเพื่อดูสเปกที่บันทึก");
    expect(client.listBuilds).not.toHaveBeenCalled();
  });

  it("shows all rows, patches metadata only and confirms deletion", async () => {
    await render();
    await click("สเปกแรก");
    expect(container.textContent).toContain("CPU A (ปิดใช้งาน)");
    expect(container.textContent).toContain("CPU B");
    expect(container.textContent).toContain("DDR4");
    expect(container.textContent).toContain("DDR5");
    expect(container.textContent).toContain("ต้องการ 2 ชิ้น · มีแล้ว 1 ชิ้น");
    await act(async () => container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
    expect(client.updateBuildDetails).toHaveBeenCalledWith("one", "สเปกแรก", "");
    expect(container.textContent).toContain("CPU A");
    expect(container.textContent).toContain("DDR5");
    window.confirm = vi.fn().mockReturnValueOnce(false).mockReturnValueOnce(true);
    await click("ลบสเปก");
    expect(client.deleteBuild).not.toHaveBeenCalled();
    await click("ลบสเปก");
    expect(client.deleteBuild).toHaveBeenCalledWith("one");
    expect(container.textContent).toContain("ลบสเปกแล้ว");
  });

  it("paginates and validates all original write fields without summaries", async () => {
    validateBuild.mockResolvedValue({ total_price_thb: 11500 });
    await render();
    await click("ถัดไป");
    expect(client.listBuilds).toHaveBeenCalledWith(2);
    await click("สเปกแรก");
    await click("ตรวจราคาปัจจุบัน");
    expect(validateBuild).toHaveBeenCalledWith({
      items: [{ product_id: "one", quantity: 2, owned_quantity: 1 }, { product_id: "two", quantity: 1, owned_quantity: 0 }],
      ram_items: [{ ram_model_id: "ram1", required_modules: 2, owned_modules: 1 }, { ram_model_id: "ram2", required_modules: 4, owned_modules: 4 }],
    });
    expect(container.textContent).toContain("11,500 บาท");
    validateBuild.mockRejectedValueOnce(new Error("ไม่สามารถตรวจได้"));
    await click("ตรวจราคาปัจจุบัน");
    expect(container.textContent).not.toContain("11,500 บาท");
    expect(container.textContent).toContain("ยังตรวจราคาไม่ได้: ไม่สามารถตรวจได้");
  });

  it("does not validate unavailable or unsafe quantities", async () => {
    client.getBuild.mockResolvedValueOnce({
      ...draft, items: [{ ...draft.items[0], quantity: Number.MAX_SAFE_INTEGER + 1 }],
    });
    await render();
    await click("สเปกแรก");
    expect(container.textContent).toContain("เกินช่วงที่รองรับ");
    expect(Array.from(container.querySelectorAll("button")).find((button) => button.textContent.includes("ตรวจราคาปัจจุบัน")).disabled).toBe(true);
    expect(validateBuild).not.toHaveBeenCalled();
  });

  it("reports unavailable items without hiding them or requesting a price", async () => {
    client.getBuild.mockResolvedValueOnce({
      ...draft, availability: { ready_for_validation: false, issues: [{ code: "PRODUCT_INACTIVE", message: "สินค้าไม่พร้อม" }] },
    });
    await render();
    await click("สเปกแรก");
    expect(container.textContent).toContain("สินค้าไม่พร้อม");
    expect(container.textContent).toContain("CPU A");
    expect(Array.from(container.querySelectorAll("button")).find((button) => button.textContent.includes("ตรวจราคาปัจจุบัน")).disabled).toBe(true);
    expect(validateBuild).not.toHaveBeenCalled();
  });

  it("does not display a delayed price after logout and login", async () => {
    let resolvePrice;
    validateBuild.mockReturnValue(new Promise((resolve) => { resolvePrice = resolve; }));
    await render();
    await click("สเปกแรก");
    act(() => Array.from(container.querySelectorAll("button")).find((button) => button.textContent.includes("ตรวจราคาปัจจุบัน")).click());
    useAuth.mockReturnValue({ authenticated: false, client });
    await render();
    await act(async () => resolvePrice({ total_price_thb: 11500 }));
    useAuth.mockReturnValue({ authenticated: true, client });
    await render();
    await click("สเปกแรก");
    expect(container.textContent).not.toContain("11,500 บาท");
  });
});
