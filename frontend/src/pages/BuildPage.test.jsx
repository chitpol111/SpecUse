import React, { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import App from "../App";
import { fetchProducts, validateBuild } from "../api/specUseApi";

vi.mock("../api/specUseApi", () => ({
  fetchProducts: vi.fn(),
  validateBuild: vi.fn(),
}));

const products = [
  { id: "cpu", name: "CPU A", category: "cpu", brand: "A", price_thb: 1000, specs: { cpu_socket: "AM4" } },
  { id: "board", name: "Board A", category: "motherboard", brand: "A", price_thb: 2000, specs: { cpu_socket: "AM4", ram_type: "DDR4" } },
  { id: "ram", ram_model_id: "ram-model", name: "RAM A", category: "ram", brand: "A", price_thb: 500, specs: { ram_type: "DDR4", modules_per_package: 1 } },
  { id: "gpu", name: "GPU A", category: "gpu", brand: "A", price_thb: 3000, specs: {} },
  { id: "gpu2", name: "GPU B", category: "gpu", brand: "B", price_thb: 3500, specs: {} },
  { id: "storage", name: "SSD A", category: "storage", brand: "A", price_thb: 4000, specs: {} },
  { id: "psu", name: "PSU A", category: "psu", brand: "A", price_thb: 5000, specs: {} },
  { id: "case", name: "Case A", category: "case", brand: "A", price_thb: 6000, specs: {} },
];

function validationResult(build, status = "unknown") {
  const items = build.items.map((item) => {
    const product = products.find((entry) => entry.id === item.product_id);
    const purchase_quantity = item.quantity - item.owned_quantity;
    return {
      ...item,
      purchase_quantity,
      unit_price_thb: product.price_thb,
      purchase_price_thb: purchase_quantity * product.price_thb,
    };
  });
  return {
    compatible: status === "passed",
    issues: status === "failed" ? [{ code: "MISMATCH", message: "พบสเปกไม่ตรงกัน" }] : [],
    items,
    ram_selections: [],
    total_price_thb: items.reduce((total, item) => total + item.purchase_price_thb, 0),
    compatibility: {
      mode: "gpu_storage_v1",
      status,
      checks: [
        { code: "CPU_SOCKET", status: status === "failed" ? "failed" : "passed", message: "ตรวจ CPU socket" },
        { code: "SYSTEM_PSU", status: "unknown", message: "ยังไม่รองรับการตรวจ PSU" },
      ],
    },
  };
}

describe("build page extended components", () => {
  let container;
  let root;

  beforeEach(() => {
    global.IS_REACT_ACT_ENVIRONMENT = true;
    window.scrollTo = vi.fn();
    fetchProducts.mockResolvedValue(products);
    validateBuild.mockImplementation(async (build) => validationResult(build));
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

  async function render() {
    await act(async () => root.render(
      <MemoryRouter initialEntries={["/build"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <App />
      </MemoryRouter>,
    ));
  }

  async function choose(category, name) {
    const nav = container.querySelector('nav[aria-label="หมวดอุปกรณ์"]');
    await act(async () => Array.from(nav.querySelectorAll("button"))
      .find((button) => button.textContent.includes(category)).click());
    const article = Array.from(container.querySelectorAll("article"))
      .find((item) => item.querySelector("h3")?.textContent === name);
    await act(async () => article.querySelector("button").click());
  }

  async function openSummary() {
    await act(async () => Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent.includes("สร้างชุดสเปก ·")).click());
  }

  it("prices each extra category through the backend and resets owned status when replacing a GPU", async () => {
    await render();
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    await choose("การ์ดจอ", "GPU A");
    await choose("พื้นที่จัดเก็บ", "SSD A");
    await choose("พาวเวอร์ซัพพลาย", "PSU A");
    await choose("เคส", "Case A");

    expect(validateBuild.mock.lastCall[0].items).toEqual(
      products.filter((product) => product.id !== "gpu2" && product.category !== "ram").map((product) => ({
        product_id: product.id,
        quantity: 1,
        owned_quantity: 0,
      })),
    );
    expect(container.textContent).toContain("21,000 ฿");
    expect(container.textContent).toContain("การตรวจพื้นฐานผ่าน");
    expect(container.textContent).not.toContain("เข้ากันได้ทั้งหมด");

    await openSummary();
    expect(container.textContent).not.toContain("ยังไม่รองรับการตรวจ PSU");
    expect(container.textContent).not.toContain("แชร์");
    expect(container.textContent).toContain("สเปกที่บันทึกเป็นส่วนตัว");
    expect(container.textContent).toContain("ตรวจพื้นฐานเฉพาะ CPU socket และชนิด RAM ที่เลือก");
    expect(container.textContent).toContain("CPU กับเมนบอร์ด — ผ่าน");
    const gpuSelect = Array.from(container.querySelectorAll("label"))
      .find((label) => label.textContent.includes("GPU")).querySelector("select");
    await act(async () => {
      gpuSelect.value = "1";
      gpuSelect.dispatchEvent(new Event("change", { bubbles: true }));
    });
    expect(validateBuild.mock.lastCall[0].items.find((item) => item.product_id === "gpu").owned_quantity).toBe(1);
    expect(container.textContent).toContain("18,000 ฿");

    await act(async () => Array.from(container.querySelectorAll("button"))
      .find((button) => button.textContent.includes("กลับไปเลือกอุปกรณ์")).click());
    await choose("การ์ดจอ", "GPU B");
    expect(validateBuild.mock.lastCall[0].items.find((item) => item.product_id === "gpu2").owned_quantity).toBe(0);
    expect(container.textContent).toContain("21,500 ฿");
  });

  it("shows a real conflict instead of treating unknown compatibility as a conflict", async () => {
    validateBuild.mockImplementation(async (build) => validationResult(build, "failed"));
    await render();
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    await openSummary();
    expect(container.textContent).toContain("พบอุปกรณ์ที่ไม่เข้ากัน");
    expect(container.textContent).toContain("พบสเปกไม่ตรงกัน");
  });

  it("does not hide a confirmed conflict outside the basic checks", async () => {
    validateBuild.mockImplementation(async (build) => {
      const result = validationResult(build, "failed");
      result.issues = [{ code: "STORAGE_INTERFACE_MISMATCH", message: "พอร์ต SATA ไม่พอ" }];
      result.compatibility.checks = [
        { code: "CPU_SOCKET", status: "passed", message: "ตรวจ CPU socket" },
        { code: "STORAGE_MOTHERBOARD", status: "failed", message: "พอร์ต SATA ไม่พอ" },
        { code: "SYSTEM_PSU", status: "unknown", message: "ยังไม่รองรับการตรวจ PSU" },
      ];
      return result;
    });
    await render();
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    await openSummary();
    expect(container.textContent).toContain("พบอุปกรณ์ที่ไม่เข้ากัน");
    expect(container.textContent).toContain("พอร์ต SATA ไม่พอ");
    expect(container.textContent).not.toContain("ยังไม่รองรับการตรวจ PSU");
  });

  it("does not say basic checks passed when the CPU check is missing", async () => {
    validateBuild.mockImplementation(async (build) => {
      const result = validationResult(build);
      result.compatibility.checks = [{ code: "SYSTEM_PSU", status: "unknown", message: "ยังตรวจไม่ได้" }];
      return result;
    });
    await render();
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    expect(container.textContent).toContain("ยังตรวจพื้นฐานไม่ได้");
    expect(container.textContent).not.toContain("การตรวจพื้นฐานผ่าน");
  });

  it("requires a RAM check before saying the selected RAM passed", async () => {
    await render();
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    await choose("หน่วยความจำ", "DDR4 RAM");
    expect(container.textContent).toContain("ยังตรวจพื้นฐานไม่ได้");
    expect(container.textContent).not.toContain("การตรวจพื้นฐานผ่าน");

    validateBuild.mockImplementation(async (build) => {
      const result = validationResult(build);
      result.compatibility.checks.push({ code: "RAM_TYPE", status: "passed", message: "ชนิด RAM ตรงกัน" });
      return result;
    });
    await choose("การ์ดจอ", "GPU A");
    await openSummary();
    expect(container.textContent).toContain("การตรวจพื้นฐานผ่าน");
    expect(container.textContent).toContain("RAM กับเมนบอร์ด — ผ่าน: ชนิด RAM ตรงกัน");
  });

  it("rejects a response without the extended compatibility contract", async () => {
    validateBuild.mockResolvedValue({
      compatible: true, issues: [], items: [], ram_selections: [], total_price_thb: 0,
    });
    await render();
    await choose("ซีพียู", "CPU A");
    await choose("เมนบอร์ด", "Board A");
    expect(container.textContent).toContain("ผลตรวจจาก Backend ไม่ตรงกับ API contract");
    expect(container.textContent).not.toContain("รายการที่ตรวจผ่าน");
  });
});
