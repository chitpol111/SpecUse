import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import SiteFooter from "../components/SiteFooter";
import SiteHeader from "../components/SiteHeader";
import { fetchProducts, validateBuild } from "../api/specUseApi";

const CATEGORIES = [
  { id: "cpu", label: "ซีพียู", shortLabel: "CPU", mark: "CPU" },
  { id: "motherboard", label: "เมนบอร์ด", shortLabel: "Mainboard", mark: "MB" },
  { id: "gpu", label: "การ์ดจอ", shortLabel: "GPU", mark: "GPU" },
  { id: "ram", label: "หน่วยความจำ", shortLabel: "RAM", mark: "RAM" },
  { id: "storage", label: "พื้นที่จัดเก็บ", shortLabel: "Storage", mark: "SSD" },
  { id: "psu", label: "พาวเวอร์ซัพพลาย", shortLabel: "Power Supply", mark: "PSU" },
  { id: "case", label: "เคส", shortLabel: "Case", mark: "CASE" },
];

const VALIDATED_CATEGORIES = new Set(["cpu", "motherboard", "ram"]);

const formatTHB = (amount) =>
  `${new Intl.NumberFormat("th-TH").format(amount)} ฿`;

const productDetails = (product) => {
  const specs = product.specs || {};
  const details = [];
  if (specs.cpu_socket) details.push(`Socket ${specs.cpu_socket}`);
  if (specs.ram_type) details.push(specs.ram_type);
  if (specs.modules_per_package) {
    details.push(`${specs.modules_per_package} โมดูล/แพ็ก`);
  }
  return details;
};

function CategoryIcon({ category, className = "h-5 w-5" }) {
  const icon = {
    cpu: (
      <>
        <rect x="5" y="5" width="14" height="14" rx="2" />
        <rect x="8" y="8" width="8" height="8" rx="1" />
        <path d="M9 1v4m6-4v4M9 19v4m6-4v4M1 9h4m-4 6h4m14-6h4m-4 6h4" />
      </>
    ),
    motherboard: (
      <>
        <rect x="4" y="3" width="16" height="18" rx="2" />
        <rect x="7" y="6" width="6" height="6" rx="1" />
        <circle cx="16.5" cy="8" r="1.5" />
        <path d="M7 16h10m-10 3h7" />
      </>
    ),
    gpu: (
      <>
        <rect x="2" y="6" width="20" height="12" rx="2" />
        <circle cx="9" cy="12" r="3" />
        <path d="M17 9v6m-15-2H1m4 7v2m13-2v2" />
      </>
    ),
    ram: (
      <>
        <rect x="2" y="8" width="20" height="8" rx="1" />
        <path d="M6 8v4m4-4v4m4-4v4m4-4v4M6 16v2m4-2v2m4-2v2m4-2v2" />
      </>
    ),
    storage: (
      <>
        <rect x="5" y="3" width="14" height="18" rx="2" />
        <circle cx="12" cy="15" r="2" />
        <path d="M9 7h6m-6 3h6" />
      </>
    ),
    psu: (
      <>
        <rect x="3" y="5" width="18" height="14" rx="2" />
        <circle cx="10" cy="12" r="4" />
        <path d="M10 8v2m0 4v2m-4-4h2m4 0h2m6-4v8" />
      </>
    ),
    case: (
      <>
        <rect x="6" y="2" width="12" height="20" rx="2" />
        <path d="M9 6h6m-6 3h6m-6 9h.01" />
      </>
    ),
  }[category];

  return (
    <svg
      aria-hidden="true"
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {icon}
    </svg>
  );
}

function FixtureNotice() {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-ruby-400/20 bg-ruby-950/40 px-4 py-3 text-sm leading-6 text-ruby-100">
      <svg
        aria-hidden="true"
        className="mt-0.5 h-5 w-5 shrink-0 text-ruby-300"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="9" />
        <path d="M12 11v5m0-8h.01" />
      </svg>
      <p>
        ตอนนี้กำลังใช้สินค้า <strong>[LOCAL TEST]</strong> จาก Backend ในเครื่อง
        ราคานี้เป็นข้อมูลทดลอง ยังไม่ใช่ catalog จริง และระบบบันทึกสเปกยังไม่พร้อมใช้งาน
      </p>
    </div>
  );
}

function ErrorNotice({ children }) {
  return (
    <div
      role="alert"
      className="rounded-xl border border-ruby-400/30 bg-ruby-950/50 p-4 text-sm leading-6 text-ruby-100"
    >
      {children}
    </div>
  );
}

function ProductArtwork({ category }) {
  const selectedCategory = CATEGORIES.find((item) => item.id === category);
  return (
    <div className="flex h-32 items-center justify-center border-b border-white/5 bg-[#18090a] sm:h-36">
      <div className="flex h-20 w-20 flex-col items-center justify-center rounded-xl border border-ruby-300/15 bg-[#210c0e] text-ruby-200">
        <CategoryIcon category={category} className="h-8 w-8" />
        <span className="mt-1 text-[10px] font-bold tracking-wider">
          {selectedCategory?.mark}
        </span>
      </div>
    </div>
  );
}

function ProductCard({ category, item, selected, onSelect }) {
  const title = item.kind === "ram" ? `${item.ramType} RAM` : item.name;
  const detailLines =
    item.kind === "ram"
      ? item.packages.map(
          (product) =>
            `${product.name} · ${product.specs.modules_per_package} โมดูล/แพ็ก · ${formatTHB(product.price_thb)}`,
        )
      : productDetails(item);
  const displayPrice =
    item.kind === "ram"
      ? `${formatTHB(Math.min(...item.packages.map((product) => product.price_thb)))} / แพ็ก`
      : formatTHB(item.price_thb);

  return (
    <article
      className={`flex min-w-0 flex-col overflow-hidden rounded-xl border bg-[#150708] transition-colors ${
        selected
          ? "border-ruby-400/55"
          : "border-white/10 hover:border-ruby-300/30"
      }`}
    >
      <ProductArtwork category={category} />
      <div className="flex flex-1 flex-col p-4">
        <p className="text-xs font-semibold text-ruby-200">
          {item.kind === "ram" ? "รุ่น RAM" : item.brand}
        </p>
        <h3 className="mt-1 min-h-12 break-words font-bold leading-6 text-slate-50">
          {title}
        </h3>
        <ul className="mt-2 min-h-12 space-y-1 text-xs leading-5 text-slate-300">
          {detailLines.length > 0 ? (
            detailLines.map((detail) => <li key={detail}>{detail}</li>)
          ) : (
            <li>{item.category}</li>
          )}
        </ul>
        <div className="mt-auto pt-4">
          <p className="mb-3 text-lg font-extrabold tabular-nums text-slate-50">
            {displayPrice}
          </p>
          <button
            type="button"
            aria-pressed={selected}
            onClick={() => onSelect(item)}
            className={`min-h-11 w-full rounded-lg px-3 py-2 text-sm font-bold transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ruby-300 ${
              selected
                ? "border border-ruby-400/40 bg-ruby-950/70 text-ruby-100 hover:bg-ruby-900/70"
                : "bg-ruby-600 text-white hover:bg-ruby-500"
            }`}
          >
            {selected ? "เลือกไว้แล้ว · เปลี่ยนรายการนี้" : "＋ เพิ่มในสเปก"}
          </button>
        </div>
      </div>
    </article>
  );
}

function BuilderSummary({
  categories,
  selectedItems,
  selectedCount,
  validation,
  validationError,
  validating,
  budget,
  onBudgetChange,
  onSelectCategory,
  onOpenSummary,
}) {
  const budgetAmount = Number(budget);
  const hasBudget = Number.isFinite(budgetAmount) && budgetAmount > 0;
  const progress =
    validation && hasBudget
      ? Math.min((validation.total_price_thb / budgetAmount) * 100, 100)
      : 0;

  return (
    <aside className="glass-panel h-fit rounded-2xl p-4 sm:p-5 lg:sticky lg:top-24">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-extrabold text-slate-50">สเปกของคุณ</h2>
        <span className="rounded-full border border-white/10 px-2.5 py-1 text-xs tabular-nums text-slate-300">
          {selectedCount}/{categories.length}
        </span>
      </div>

      <ul className="mt-4 divide-y divide-white/10">
        {categories.map((category) => {
          const item = selectedItems[category.id];
          return (
            <li key={category.id} className="flex items-start gap-3 py-3 first:pt-0">
              <span
                className={`mt-0.5 rounded-lg p-2 ${
                  item ? "bg-ruby-950/70 text-ruby-200" : "bg-white/5 text-slate-500"
                }`}
              >
                <CategoryIcon category={category.id} className="h-4 w-4" />
              </span>
              <button
                type="button"
                onClick={() => onSelectCategory(category.id)}
                className="min-w-0 flex-1 text-left"
              >
                <span className="block text-xs text-slate-400">{category.label}</span>
                <span className={`mt-0.5 block truncate text-sm font-semibold ${item ? "text-slate-100" : "text-slate-500"}`}>
                  {item?.name || "ยังไม่ได้เลือก"}
                </span>
              </button>
            </li>
          );
        })}
      </ul>

      <div className="mt-2 border-t border-ruby-400/15 pt-4">
        <label htmlFor="build-budget" className="block text-sm font-semibold text-slate-200">
          งบประมาณ (บาท)
        </label>
        <input
          id="build-budget"
          type="number"
          min="1"
          inputMode="numeric"
          value={budget}
          onChange={(event) => onBudgetChange(event.target.value)}
          placeholder="เช่น 30,000"
          className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-obsidian-900 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-ruby-300/70 focus:outline-none focus:ring-2 focus:ring-ruby-300/20"
        />
        {validation && hasBudget && (
          <div className="mt-3" aria-live="polite">
            <div className="mb-1.5 flex justify-between gap-3 text-xs text-slate-300">
              <span>ราคาอุปกรณ์ที่ยังไม่มีเทียบกับงบ</span>
              <span className="tabular-nums">
                {Math.round((validation.total_price_thb / budgetAmount) * 100)}%
              </span>
            </div>
            <div
              role="progressbar"
              aria-label="สัดส่วนราคาอุปกรณ์ที่ยังไม่มีเทียบกับงบประมาณ"
              aria-valuenow={Math.min(
                Math.round((validation.total_price_thb / budgetAmount) * 100),
                100,
              )}
              aria-valuemin="0"
              aria-valuemax="100"
              className="h-2 overflow-hidden rounded-full bg-white/10"
            >
              <div
                className={`h-full rounded-full transition-[width] duration-200 ${
                  validation.total_price_thb > budgetAmount
                    ? "bg-ruby-500"
                    : "bg-emerald-500"
                }`}
                style={{ width: `${progress}%` }}
              />
            </div>
          </div>
        )}
      </div>

      <div className="mt-4 flex items-end justify-between gap-3 border-t border-white/10 pt-4">
        <span className="text-sm text-slate-300">ราคาโดยประมาณ</span>
        <span className="text-xl font-black tabular-nums text-slate-50">
          {validation ? formatTHB(validation.total_price_thb) : "—"}
        </span>
      </div>
      <p className="mt-1 text-xs leading-5 text-slate-400">
        {validating
          ? "กำลังคำนวณจาก Backend…"
          : validationError
            ? "ตรวจราคาไม่สำเร็จ"
            : validation
              ? "ราคาจากข้อมูลทดลองของ Backend"
              : "เลือก CPU และเมนบอร์ดเพื่อคำนวณราคา"}
      </p>
      {validationError && (
        <p role="alert" className="mt-2 text-xs leading-5 text-ruby-200">
          {validationError}
        </p>
      )}
      {validation && (
        <p
          className={`mt-3 text-sm font-bold ${
            validation.compatible ? "text-emerald-300" : "text-ruby-300"
          }`}
          role="status"
        >
          {validation.compatible
            ? "อุปกรณ์ที่ตรวจเข้ากันได้"
            : "พบรายการที่ไม่เข้ากัน"}
        </p>
      )}

      <button
        type="button"
        onClick={onOpenSummary}
        disabled={selectedCount === 0}
        className="mt-5 min-h-12 w-full rounded-lg bg-ruby-600 px-4 py-3 font-extrabold text-white transition hover:bg-ruby-500 disabled:cursor-not-allowed disabled:opacity-45"
      >
        สร้างชุดสเปก · ดูรายการที่เลือก
      </button>
      <p className="mt-2 text-center text-xs leading-5 text-slate-400">
        ยังไม่สามารถบันทึกสเปกได้ในต้นแบบนี้
      </p>
    </aside>
  );
}

function SummaryLine({ row }) {
  return (
    <tr className="border-b border-white/10 last:border-b-0">
      <td className="px-4 py-4 align-top">
        <span className="block font-bold text-slate-100">{row.name}</span>
        <span className="mt-1 block text-xs text-slate-400">{row.category}</span>
      </td>
      <td className="max-w-64 px-4 py-4 align-top text-sm leading-6 text-slate-300">
        {row.details}
      </td>
      <td className="whitespace-nowrap px-4 py-4 text-center text-sm tabular-nums text-slate-200">
        {row.quantity}
      </td>
      <td className="whitespace-nowrap px-4 py-4 text-center text-sm text-slate-400">
        {row.status}
      </td>
      <td className="whitespace-nowrap px-4 py-4 text-right text-sm font-bold tabular-nums text-slate-100">
        {row.total}
      </td>
    </tr>
  );
}

function BuildList({ rows }) {
  if (rows.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-white/15 px-5 py-10 text-center">
        <p className="font-bold text-slate-200">ยังไม่มีอุปกรณ์ในรายการ</p>
        <p className="mt-2 text-sm leading-6 text-slate-400">
          กลับไปเลือกอุปกรณ์จากหมวดที่ต้องการก่อน แล้วรายการจะปรากฏที่นี่
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="overflow-x-auto rounded-xl border border-white/10">
        <table className="hidden w-full min-w-[720px] border-collapse text-left md:table">
          <thead className="bg-[#1b1011] text-xs font-bold text-slate-200">
            <tr>
              <th className="px-4 py-3">อุปกรณ์</th>
              <th className="px-4 py-3">รายละเอียด</th>
              <th className="px-4 py-3 text-center">จำนวน</th>
              <th className="px-4 py-3 text-center">สถานะ</th>
              <th className="px-4 py-3 text-right">ราคารวมโดยประมาณ</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <SummaryLine key={row.id} row={row} />
            ))}
          </tbody>
        </table>
      </div>
      <ul className="space-y-3 md:hidden">
        {rows.map((row) => (
          <li
            key={row.id}
            className="rounded-xl border border-white/10 bg-[#150708] p-4"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="break-words font-bold text-slate-100">{row.name}</p>
                <p className="mt-1 text-xs text-slate-400">{row.category}</p>
              </div>
              <p className="shrink-0 text-right font-bold tabular-nums text-slate-100">
                {row.total}
              </p>
            </div>
            <p className="mt-3 text-sm leading-6 text-slate-300">{row.details}</p>
            <dl className="mt-3 grid grid-cols-2 gap-3 border-t border-white/10 pt-3 text-sm">
              <div>
                <dt className="text-xs text-slate-400">จำนวน</dt>
                <dd className="mt-1 text-slate-200">{row.quantity}</dd>
              </div>
              <div>
                <dt className="text-xs text-slate-400">สถานะ</dt>
                <dd className="mt-1 text-slate-200">{row.status}</dd>
              </div>
            </dl>
          </li>
        ))}
      </ul>
    </>
  );
}

export default function BuildPage() {
  const [products, setProducts] = useState([]);
  const [loadingProducts, setLoadingProducts] = useState(true);
  const [productsError, setProductsError] = useState("");
  const [activeCategory, setActiveCategory] = useState("cpu");
  const [selectedComponents, setSelectedComponents] = useState({
    cpu: "",
    motherboard: "",
    gpu: "",
    ram: "",
    storage: "",
    psu: "",
    case: "",
  });
  const [cpuOwned, setCpuOwned] = useState(0);
  const [motherboardOwned, setMotherboardOwned] = useState(0);
  const [requiredModules, setRequiredModules] = useState(2);
  const [ownedModules, setOwnedModules] = useState(0);
  const [search, setSearch] = useState("");
  const [sortOrder, setSortOrder] = useState("price-asc");
  const [budget, setBudget] = useState("");
  const [toast, setToast] = useState(null);
  const [showSummary, setShowSummary] = useState(false);
  const [validation, setValidation] = useState(null);
  const [validationError, setValidationError] = useState("");
  const [validating, setValidating] = useState(false);

  const loadProducts = useCallback(async (signal) => {
    setLoadingProducts(true);
    setProductsError("");
    try {
      setProducts(await fetchProducts({ signal }));
    } catch (error) {
      if (error.name !== "AbortError") {
        setProductsError(error.message);
      }
    } finally {
      if (!signal?.aborted) setLoadingProducts(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    loadProducts(controller.signal);
    return () => controller.abort();
  }, [loadProducts]);

  useEffect(() => {
    if (!toast) return undefined;
    const timeout = window.setTimeout(() => setToast(null), 4200);
    return () => window.clearTimeout(timeout);
  }, [toast]);

  const ramModels = useMemo(() => {
    const models = new Map();
    for (const product of products) {
      if (product.category !== "ram" || !product.ram_model_id) continue;
      const model = models.get(product.ram_model_id) || {
        id: product.ram_model_id,
        category: "ram",
        ramType: product.specs?.ram_type || "ไม่ระบุชนิด",
        packages: [],
        kind: "ram",
      };
      model.packages.push(product);
      models.set(product.ram_model_id, model);
    }
    return Array.from(models.values());
  }, [products]);

  const productsByID = useMemo(
    () => new Map(products.map((product) => [product.id, product])),
    [products],
  );

  const selectedItems = useMemo(() => {
    const selected = {};
    for (const category of CATEGORIES) {
      const id = selectedComponents[category.id];
      if (!id) {
        selected[category.id] = null;
      } else if (category.id === "ram") {
        const model = ramModels.find((item) => item.id === id);
        selected[category.id] = model
          ? {
              name: `${model.ramType} RAM`,
              brand: model.packages.map((item) => item.brand).join(", "),
              category: "ram",
              kind: "ram",
            }
          : null;
      } else {
        selected[category.id] = productsByID.get(id) || null;
      }
    }
    return selected;
  }, [productsByID, ramModels, selectedComponents]);

  const selectionCount = Object.values(selectedComponents).filter(Boolean).length;
  const requestPayload = useMemo(
    () => ({
      items: [
        ...(selectedComponents.cpu
          ? [{
              product_id: selectedComponents.cpu,
              quantity: 1,
              owned_quantity: cpuOwned,
            }]
          : []),
        ...(selectedComponents.motherboard
          ? [{
              product_id: selectedComponents.motherboard,
              quantity: 1,
              owned_quantity: motherboardOwned,
            }]
          : []),
      ],
      ram_items: selectedComponents.ram
        ? [{
            ram_model_id: selectedComponents.ram,
            required_modules: requiredModules,
            owned_modules: ownedModules,
          }]
        : [],
    }),
    [
      cpuOwned,
      motherboardOwned,
      ownedModules,
      requiredModules,
      selectedComponents.cpu,
      selectedComponents.motherboard,
      selectedComponents.ram,
    ],
  );

  useEffect(() => {
    if (!selectedComponents.cpu || !selectedComponents.motherboard) {
      setValidation(null);
      setValidationError("");
      setValidating(false);
      return undefined;
    }

    const controller = new AbortController();
    setValidation(null);
    setValidationError("");
    setValidating(true);

    validateBuild(requestPayload, { signal: controller.signal })
      .then((result) => {
        if (
          typeof result.compatible !== "boolean" ||
          !Array.isArray(result.issues) ||
          !Array.isArray(result.items) ||
          !Array.isArray(result.ram_selections) ||
          !Number.isInteger(result.total_price_thb)
        ) {
          throw new Error("ผลตรวจจาก Backend ไม่ตรงกับ API contract");
        }
        setValidation(result);
      })
      .catch((error) => {
        if (error.name !== "AbortError") setValidationError(error.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setValidating(false);
      });

    return () => controller.abort();
  }, [requestPayload, selectedComponents.cpu, selectedComponents.motherboard]);

  const categoryProducts = useMemo(() => {
    const category = CATEGORIES.find((item) => item.id === activeCategory);
    if (!category) return [];
    if (activeCategory === "ram") return ramModels;
    return products.filter((product) => product.category === activeCategory);
  }, [activeCategory, products, ramModels]);

  const visibleProducts = useMemo(() => {
    const normalizedSearch = search.trim().toLocaleLowerCase("th");
    const filtered = categoryProducts.filter((item) => {
      const text =
        item.kind === "ram"
          ? [
              item.ramType,
              ...item.packages.flatMap((product) => [
                product.name,
                product.brand,
              ]),
            ].join(" ")
          : [item.name, item.brand, ...productDetails(item)].join(" ");
      return text.toLocaleLowerCase("th").includes(normalizedSearch);
    });
    return [...filtered].sort((a, b) => {
      const priceA =
        a.kind === "ram"
          ? Math.min(...a.packages.map((product) => product.price_thb))
          : a.price_thb;
      const priceB =
        b.kind === "ram"
          ? Math.min(...b.packages.map((product) => product.price_thb))
          : b.price_thb;
      return sortOrder === "price-desc" ? priceB - priceA : priceA - priceB;
    });
  }, [categoryProducts, search, sortOrder]);

  const summaryRows = useMemo(() => {
    if (validation) {
      const componentRows = validation.items.map((item) => {
        const product = productsByID.get(item.product_id);
        const category = product
          ? CATEGORIES.find((entry) => entry.id === product.category)?.label
          : "อุปกรณ์";
        const detail = product
          ? productDetails(product).join(" · ") || product.brand
          : "รายละเอียดสินค้า";
        return {
          id: item.product_id,
          name: product?.name || "อุปกรณ์ที่เลือก",
          category,
          details:
            item.purchase_quantity > 0
              ? detail
              : `${detail} · มีอยู่แล้ว`,
          quantity:
            item.purchase_quantity > 0
              ? `${item.purchase_quantity} ชิ้น`
              : "0 ชิ้น",
          status: item.purchase_quantity > 0 ? "ยังไม่มี" : "มีแล้ว",
          total: formatTHB(item.purchase_price_thb),
        };
      });
      const ramRows = validation.ram_selections.flatMap((selection) => {
        if (selection.packages.length === 0) {
          return [{
            id: `ram-owned-${selection.ram_model_id}`,
            name: selectedItems.ram?.name || "RAM ที่เลือก",
            category: "RAM",
            details: `มี RAM อยู่แล้ว ${selection.owned_modules} โมดูล`,
            quantity: `${selection.owned_modules} โมดูล`,
            status: "มีแล้ว",
            total: formatTHB(selection.total_price_thb),
          }];
        }
        return selection.packages.map((pack) => {
          const product = productsByID.get(pack.product_id);
          return {
            id: `ram-${pack.product_id}`,
            name: product?.name || "แพ็ก RAM",
            category: "RAM",
            details: `${pack.modules_per_package} โมดูล/แพ็ก · จัดให้ครบ ${selection.required_modules} โมดูล`,
            quantity: `${pack.quantity} แพ็ก`,
            status: "ยังไม่มี",
            total: formatTHB(pack.unit_price_thb * pack.quantity),
          };
        });
      });
      return [...componentRows, ...ramRows];
    }

    return CATEGORIES.flatMap((category) => {
      const item = selectedItems[category.id];
      if (!item) return [];
      return [{
        id: `draft-${category.id}`,
        name: item.name,
        category: category.label,
        details:
          category.id === "ram"
            ? `ต้องการ ${requiredModules} โมดูล · ${item.brand || "รุ่น RAM ที่เลือก"}`
            : productDetails(item).join(" · ") || item.brand,
        quantity: category.id === "ram" ? `${requiredModules} โมดูล` : "1 ชิ้น",
        status:
          (category.id === "cpu" && cpuOwned > 0) ||
          (category.id === "motherboard" && motherboardOwned > 0) ||
          (category.id === "ram" && ownedModules >= requiredModules)
            ? "มีแล้ว"
            : "ยังไม่มี",
        total: "รอตรวจราคา",
      }];
    });
  }, [
    cpuOwned,
    motherboardOwned,
    ownedModules,
    productsByID,
    requiredModules,
    selectedItems,
    validation,
  ]);

  const selectedCategory = CATEGORIES.find((item) => item.id === activeCategory);
  const hasCategoryProducts = categoryProducts.length > 0;
  const categorySupported = VALIDATED_CATEGORIES.has(activeCategory);

  function chooseCategory(categoryID) {
    setActiveCategory(categoryID);
    setSearch("");
  }

  function addToBuild(item) {
    if (!categorySupported) return;
    setSelectedComponents((current) => ({
      ...current,
      [activeCategory]: item.id,
    }));
    const itemName =
      item.kind === "ram" ? `${item.ramType} RAM` : item.name;
    setToast({
      category: selectedCategory?.label || "อุปกรณ์",
      name: itemName,
    });
  }

  const openSummary = () => {
    setShowSummary(true);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const returnToBuilder = () => {
    setShowSummary(false);
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  return (
    <>
      <SiteHeader />
      <main className="relative min-h-screen overflow-hidden pb-24 lg:pb-12">
        <div className="hero-grid pointer-events-none absolute inset-x-0 top-0 h-96 opacity-20" aria-hidden="true" />
        <div className="relative mx-auto max-w-[1440px] px-4 py-7 sm:px-6 sm:py-10 xl:px-8">
          <Link
            to="/"
            className="inline-flex min-h-9 items-center text-sm font-semibold text-ruby-200 underline-offset-4 hover:text-ruby-100 hover:underline"
          >
            ← กลับหน้าแรก
          </Link>

          <div className="mt-5 flex flex-wrap items-end justify-between gap-4">
            <div>
              <h1 className="text-2xl font-black tracking-tight text-slate-50 sm:text-3xl">
                {showSummary ? "รายการสเปกที่เลือก" : "จัดชุดสเปกคอมของคุณ"}
              </h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-300 sm:text-base">
                {showSummary
                  ? "ตรวจรายการ อุปกรณ์ที่มีแล้ว และราคาโดยประมาณจากข้อมูลที่ Backend คำนวณให้"
                  : "เลือกอุปกรณ์ทีละหมวด แล้วตรวจความเข้ากันได้และค่าใช้จ่ายโดยประมาณ"}
              </p>
            </div>
            {showSummary && (
              <button
                type="button"
                onClick={returnToBuilder}
                className="min-h-10 rounded-lg border border-white/15 px-4 py-2 text-sm font-bold text-slate-100 transition hover:border-ruby-300/40 hover:text-ruby-100"
              >
                ← กลับไปเลือกอุปกรณ์
              </button>
            )}
          </div>

          <div className="mt-6">
            <FixtureNotice />
          </div>

          {toast && !showSummary && (
            <div
              role="status"
              aria-live="polite"
              className="fixed right-4 top-20 z-50 flex max-w-[calc(100vw-2rem)] items-start gap-3 rounded-xl border border-emerald-400/30 bg-[#10251d] px-4 py-3 text-sm text-emerald-100 shadow-[0_12px_36px_rgba(0,0,0,0.4)]"
            >
              <svg
                aria-hidden="true"
                className="mt-0.5 h-5 w-5 shrink-0 text-emerald-300"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="m5 12 4 4L19 6" />
              </svg>
              <p className="leading-5">
                เพิ่ม <strong>{toast.name}</strong> ในหมวด{toast.category}แล้ว
              </p>
              <button
                type="button"
                aria-label="ปิดข้อความแจ้งเตือน"
                onClick={() => setToast(null)}
                className="ml-1 rounded p-1 text-emerald-100/70 hover:bg-white/10 hover:text-white"
              >
                ×
              </button>
            </div>
          )}

          {loadingProducts && (
            <div role="status" className="mt-7 grid gap-6 lg:grid-cols-[200px_minmax(0,1fr)_290px]">
              <div className="hidden space-y-2 lg:block">
                {CATEGORIES.map((category) => (
                  <div
                    key={category.id}
                    className="h-12 animate-pulse rounded-lg bg-white/5"
                  />
                ))}
              </div>
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {[1, 2, 3].map((item) => (
                  <div
                    key={item}
                    className="h-72 animate-pulse rounded-xl border border-white/5 bg-white/[0.03]"
                  />
                ))}
              </div>
              <div className="hidden h-96 animate-pulse rounded-2xl bg-white/[0.03] lg:block" />
              <p className="sr-only">กำลังโหลดรายการสินค้าจาก Backend…</p>
            </div>
          )}

          {productsError && (
            <div className="mt-7 max-w-2xl">
              <ErrorNotice>{productsError}</ErrorNotice>
              <button
                type="button"
                onClick={() => loadProducts()}
                className="mt-3 min-h-11 rounded-lg bg-ruby-600 px-5 py-2 text-sm font-bold text-white hover:bg-ruby-500"
              >
                ลองโหลดสินค้าอีกครั้ง
              </button>
            </div>
          )}

          {!loadingProducts && !productsError && products.length === 0 && (
            <div className="mt-7 max-w-2xl">
              <ErrorNotice>Backend ยังไม่มีสินค้า active ให้ทดลอง เลือกตรวจสอบข้อมูล fixture แล้วลองใหม่</ErrorNotice>
              <button
                type="button"
                onClick={() => loadProducts()}
                className="mt-3 min-h-11 rounded-lg bg-ruby-600 px-5 py-2 text-sm font-bold text-white hover:bg-ruby-500"
              >
                โหลดสินค้าอีกครั้ง
              </button>
            </div>
          )}

          {!loadingProducts && !productsError && products.length > 0 && (
            <>
              {showSummary ? (
                <section className="mt-7 grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_320px]">
                  <div className="glass-panel min-w-0 rounded-2xl p-4 sm:p-6">
                    <div className="flex flex-wrap items-start justify-between gap-4">
                      <div>
                        <h2 className="text-xl font-extrabold text-slate-50">รายการอุปกรณ์</h2>
                        <p className="mt-1 text-sm leading-6 text-slate-300">
                          ชื่ออุปกรณ์ รายละเอียด จำนวน สถานะ และราคารวมโดยประมาณ
                        </p>
                      </div>
                      <span className="rounded-full border border-white/10 px-3 py-1 text-sm text-slate-300">
                        {summaryRows.length} รายการ
                      </span>
                    </div>

                    {validation && (
                      <div
                        className={`mt-5 rounded-xl border p-4 ${
                          validation.compatible
                            ? "border-emerald-400/25 bg-emerald-950/25"
                            : "border-ruby-400/30 bg-ruby-950/30"
                        }`}
                        role="status"
                      >
                        <p
                          className={`font-bold ${
                            validation.compatible
                              ? "text-emerald-200"
                              : "text-ruby-200"
                          }`}
                        >
                          {validation.compatible
                            ? "เข้ากันได้ทั้งหมด"
                            : "พบอุปกรณ์ที่อาจไม่เข้ากัน"}
                        </p>
                        {validation.issues.length > 0 && (
                          <ul className="mt-2 list-inside list-disc space-y-1 text-sm leading-6 text-ruby-100">
                            {validation.issues.map((issue) => (
                              <li key={issue.code}>{issue.message}</li>
                            ))}
                          </ul>
                        )}
                      </div>
                    )}
                    {validating && (
                      <p role="status" className="mt-5 text-sm text-slate-300">
                        Backend กำลังตรวจความเข้ากันได้และคำนวณราคาโดยประมาณ…
                      </p>
                    )}
                    {validationError && (
                      <div className="mt-5">
                        <ErrorNotice>{validationError}</ErrorNotice>
                      </div>
                    )}
                    {!selectedComponents.cpu || !selectedComponents.motherboard ? (
                      <p className="mt-5 rounded-lg border border-amber-300/20 bg-amber-950/20 p-3 text-sm leading-6 text-amber-100">
                        เลือก CPU และเมนบอร์ดก่อน จึงจะตรวจความเข้ากันได้และคำนวณราคาโดยประมาณได้
                      </p>
                    ) : null}

                    <div className="mt-5">
                      <BuildList rows={summaryRows} />
                    </div>
                    <p className="mt-3 text-xs leading-5 text-slate-400">
                      ราคาเป็นข้อมูลตัวอย่างจาก Backend ใช้เพื่อวางแผนเท่านั้น และคำนวณเฉพาะอุปกรณ์ที่ยังไม่มีในชุด
                    </p>
                  </div>

                  <div className="glass-panel rounded-2xl p-5">
                    <h2 className="text-lg font-extrabold text-slate-50">สรุปยอด</h2>
                    <div className="mt-4 flex items-center justify-between gap-4 border-b border-white/10 pb-4 text-sm">
                      <span className="text-slate-300">อุปกรณ์ที่เลือก</span>
                      <span className="tabular-nums text-slate-200">{summaryRows.length} รายการ</span>
                    </div>
                    <div className="flex items-end justify-between gap-3 py-4">
                      <span className="font-bold text-slate-100">ราคาอุปกรณ์ที่ยังไม่มี</span>
                      <span className="text-2xl font-black tabular-nums text-ruby-200">
                        {validation
                          ? formatTHB(validation.total_price_thb)
                          : validating
                            ? "กำลังคำนวณ"
                            : "—"}
                      </span>
                    </div>
                    <fieldset className="border-t border-white/10 pt-4">
                      <legend className="text-sm font-bold text-slate-100">
                        อุปกรณ์ที่มีอยู่แล้ว
                      </legend>
                      <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                        <label className="text-xs text-slate-300">
                          CPU
                          <select
                            value={cpuOwned}
                            onChange={(event) => setCpuOwned(Number(event.target.value))}
                            disabled={!selectedComponents.cpu || validating}
                            className="mt-1 min-h-10 w-full rounded-lg border border-white/10 bg-obsidian-900 px-3 text-sm text-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            <option value={0}>ยังไม่มี</option>
                            <option value={1}>มีอยู่แล้ว</option>
                          </select>
                        </label>
                        <label className="text-xs text-slate-300">
                          เมนบอร์ด
                          <select
                            value={motherboardOwned}
                            onChange={(event) => setMotherboardOwned(Number(event.target.value))}
                            disabled={!selectedComponents.motherboard || validating}
                            className="mt-1 min-h-10 w-full rounded-lg border border-white/10 bg-obsidian-900 px-3 text-sm text-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            <option value={0}>ยังไม่มี</option>
                            <option value={1}>มีอยู่แล้ว</option>
                          </select>
                        </label>
                      </div>
                      {selectedComponents.ram && (
                        <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                          <label className="text-xs text-slate-300">
                            โมดูล RAM ที่ต้องการ
                            <input
                              type="number"
                              min="1"
                              step="1"
                              value={requiredModules}
                              disabled={validating}
                              onChange={(event) => {
                                const nextRequired = Math.max(1, Number(event.target.value) || 1);
                                setRequiredModules(nextRequired);
                                setOwnedModules((current) => Math.min(current, nextRequired));
                              }}
                              className="mt-1 min-h-10 w-full rounded-lg border border-white/10 bg-obsidian-900 px-3 text-sm text-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                            />
                          </label>
                          <label className="text-xs text-slate-300">
                            โมดูล RAM ที่มีแล้ว
                            <input
                              type="number"
                              min="0"
                              max={requiredModules}
                              step="1"
                              value={ownedModules}
                              disabled={validating}
                              onChange={(event) => {
                                setOwnedModules(
                                  Math.min(
                                    requiredModules,
                                    Math.max(0, Number(event.target.value) || 0),
                                  ),
                                );
                              }}
                              className="mt-1 min-h-10 w-full rounded-lg border border-white/10 bg-obsidian-900 px-3 text-sm text-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                            />
                          </label>
                        </div>
                      )}
                      <p className="mt-3 text-xs leading-5 text-slate-400">
                        ปรับจำนวนที่มีแล้วเพื่อให้ Backend คำนวณราคาอุปกรณ์ที่ยังไม่มีใหม่
                      </p>
                    </fieldset>
                    <label htmlFor="summary-budget" className="block text-sm font-semibold text-slate-200">
                      งบประมาณ (บาท)
                    </label>
                    <input
                      id="summary-budget"
                      type="number"
                      min="1"
                      inputMode="numeric"
                      value={budget}
                      onChange={(event) => setBudget(event.target.value)}
                      placeholder="ใส่งบที่ต้องการ"
                      className="mt-2 min-h-11 w-full rounded-lg border border-white/15 bg-obsidian-900 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-ruby-300/70 focus:outline-none focus:ring-2 focus:ring-ruby-300/20"
                    />
                    {validation && Number(budget) > 0 && (
                      <p className={`mt-2 text-sm ${validation.total_price_thb > Number(budget) ? "text-ruby-200" : "text-emerald-200"}`}>
                        {validation.total_price_thb > Number(budget)
                          ? `เกินงบ ${formatTHB(validation.total_price_thb - Number(budget))}`
                          : `เหลืองบ ${formatTHB(Number(budget) - validation.total_price_thb)}`}
                      </p>
                    )}
                    <div className="mt-5 grid gap-2">
                      <button
                        type="button"
                        disabled
                        aria-describedby="build-actions-note"
                        className="min-h-11 cursor-not-allowed rounded-lg bg-ruby-700/50 px-4 py-2 text-sm font-bold text-white/60"
                      >
                        บันทึกชุดสเปก
                      </button>
                      <button
                        type="button"
                        disabled
                        aria-describedby="build-actions-note"
                        className="min-h-11 cursor-not-allowed rounded-lg border border-white/10 px-4 py-2 text-sm font-bold text-slate-400"
                      >
                        แชร์รายการนี้
                      </button>
                    </div>
                    <p id="build-actions-note" className="mt-3 text-xs leading-5 text-slate-400">
                      บันทึกและแชร์ยังไม่เปิดใช้ในต้นแบบนี้
                    </p>
                  </div>
                </section>
              ) : (
                <div className="mt-7 grid items-start gap-5 lg:grid-cols-[190px_minmax(0,1fr)_300px] xl:gap-6">
                  <nav aria-label="หมวดอุปกรณ์" className="hidden lg:block">
                    <p className="mb-2 px-3 text-xs font-bold tracking-wide text-slate-400">
                      เลือกหมวดอุปกรณ์
                    </p>
                    <ul className="space-y-1.5">
                      {CATEGORIES.map((category) => {
                        const selected = Boolean(selectedComponents[category.id]);
                        const active = activeCategory === category.id;
                        const count =
                          category.id === "ram"
                            ? ramModels.length
                            : products.filter((product) => product.category === category.id).length;
                        return (
                          <li key={category.id}>
                            <button
                              type="button"
                              onClick={() => chooseCategory(category.id)}
                              aria-current={active ? "page" : undefined}
                              className={`flex min-h-11 w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm transition ${
                                active
                                  ? "bg-ruby-700/70 font-bold text-white"
                                  : "text-slate-300 hover:bg-white/5 hover:text-white"
                              }`}
                            >
                              <CategoryIcon category={category.id} className="h-4 w-4 shrink-0" />
                              <span className="min-w-0 flex-1 truncate">{category.label}</span>
                              <span className="text-xs tabular-nums text-slate-400">
                                {selected ? "✓" : count || ""}
                              </span>
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  </nav>

                  <section className="min-w-0" aria-labelledby="catalog-heading">
                    <div className="mb-3 overflow-x-auto lg:hidden">
                      <div className="flex min-w-max gap-2 pb-2">
                        {CATEGORIES.map((category) => (
                          <button
                            key={category.id}
                            type="button"
                            onClick={() => chooseCategory(category.id)}
                            aria-pressed={activeCategory === category.id}
                            className={`min-h-10 rounded-lg border px-3 text-sm font-semibold ${
                              activeCategory === category.id
                                ? "border-ruby-400/40 bg-ruby-800/60 text-white"
                                : "border-white/10 bg-white/[0.03] text-slate-300"
                            }`}
                          >
                            {category.shortLabel}
                          </button>
                        ))}
                      </div>
                    </div>

                    <div className="glass-panel rounded-2xl p-4 sm:p-5">
                      <div className="flex flex-wrap items-end justify-between gap-3">
                        <div>
                          <h2 id="catalog-heading" className="text-lg font-extrabold text-slate-50">
                            {selectedCategory?.label}
                          </h2>
                          <p className="mt-1 text-xs text-slate-400">
                            {categoryProducts.length} รายการในข้อมูลตัวอย่าง
                          </p>
                        </div>
                        <label className="min-w-[145px] text-xs text-slate-400">
                          เรียงราคา
                          <select
                            value={sortOrder}
                            onChange={(event) => setSortOrder(event.target.value)}
                            className="mt-1 min-h-10 w-full rounded-lg border border-white/10 bg-obsidian-900 px-3 text-sm text-slate-100 focus:border-ruby-300/70 focus:outline-none focus:ring-2 focus:ring-ruby-300/20"
                          >
                            <option value="price-asc">ราคาต่ำไปสูง</option>
                            <option value="price-desc">ราคาสูงไปต่ำ</option>
                          </select>
                        </label>
                      </div>
                      <label className="mt-4 flex min-h-11 items-center gap-2 rounded-lg border border-white/10 bg-obsidian-900 px-3 text-sm text-slate-400 focus-within:border-ruby-300/60 focus-within:ring-2 focus-within:ring-ruby-300/15">
                        <svg
                          aria-hidden="true"
                          className="h-4 w-4 shrink-0"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="1.8"
                          strokeLinecap="round"
                        >
                          <circle cx="11" cy="11" r="7" />
                          <path d="m16 16 4 4" />
                        </svg>
                        <span className="sr-only">ค้นหาสินค้า</span>
                        <input
                          type="search"
                          value={search}
                          onChange={(event) => setSearch(event.target.value)}
                          placeholder={`ค้นหา${selectedCategory?.label || "สินค้า"}`}
                          className="min-w-0 flex-1 bg-transparent py-2 text-slate-100 placeholder:text-slate-500 focus:outline-none"
                        />
                      </label>
                    </div>

                    {!categorySupported ? (
                      <div className="mt-4 rounded-xl border border-dashed border-white/15 bg-white/[0.02] px-5 py-9 text-center">
                        <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-white/5 text-slate-400">
                          <CategoryIcon category={activeCategory} className="h-6 w-6" />
                        </span>
                        <h3 className="mt-4 font-bold text-slate-200">
                          หมวดนี้ยังไม่มีข้อมูลสำหรับเลือก
                        </h3>
                        <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-slate-400">
                          Backend รุ่นปัจจุบันรองรับการตรวจ CPU, เมนบอร์ด และ RAM
                          หมวดอื่นจะแสดงเมื่อมี catalog และ API รองรับ
                        </p>
                      </div>
                    ) : !hasCategoryProducts ? (
                      <div className="mt-4 rounded-xl border border-dashed border-white/15 bg-white/[0.02] px-5 py-9 text-center">
                        <span className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-white/5 text-slate-400">
                          <CategoryIcon category={activeCategory} className="h-6 w-6" />
                        </span>
                        <h3 className="mt-4 font-bold text-slate-200">
                          ยังไม่มี{selectedCategory?.label}ใน catalog ตัวอย่าง
                        </h3>
                        <p className="mx-auto mt-2 max-w-md text-sm leading-6 text-slate-400">
                          รายการสินค้าจะปรากฏที่นี่เมื่อ Backend มีข้อมูลในหมวดนี้
                        </p>
                      </div>
                    ) : visibleProducts.length === 0 ? (
                      <p className="mt-4 rounded-xl border border-white/10 p-6 text-center text-sm text-slate-300">
                        ไม่พบสินค้าที่ตรงกับคำค้น
                      </p>
                    ) : (
                      <>
                        {!categorySupported && (
                          <div className="mt-4">
                            <ErrorNotice>
                              หมวดนี้ยังไม่รองรับการตรวจและคำนวณราคาจาก Backend
                            </ErrorNotice>
                          </div>
                        )}
                        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                          {visibleProducts.map((item) => (
                            <ProductCard
                              key={item.id}
                              category={activeCategory}
                              item={item}
                              selected={selectedComponents[activeCategory] === item.id}
                              onSelect={addToBuild}
                            />
                          ))}
                        </div>
                      </>
                    )}
                  </section>

                  <BuilderSummary
                    categories={CATEGORIES}
                    selectedItems={selectedItems}
                    selectedCount={selectionCount}
                    validation={validation}
                    validationError={validationError}
                    validating={validating}
                    budget={budget}
                    onBudgetChange={setBudget}
                    onSelectCategory={chooseCategory}
                    onOpenSummary={openSummary}
                  />
                </div>
              )}
            </>
          )}
        </div>
      </main>
      {!loadingProducts && !productsError && products.length > 0 && !showSummary && (
        <div className="fixed inset-x-0 bottom-0 z-40 border-t border-white/10 bg-[#0b0202]/95 px-4 py-3 backdrop-blur lg:hidden">
          <button
            type="button"
            onClick={openSummary}
            disabled={selectionCount === 0}
            className="min-h-11 w-full rounded-lg bg-ruby-600 px-4 py-2 text-sm font-extrabold text-white disabled:cursor-not-allowed disabled:opacity-45"
          >
            {`สร้างชุดสเปก (${selectionCount})`}
            {validation ? ` · ${formatTHB(validation.total_price_thb)}` : ""}
          </button>
        </div>
      )}
      <SiteFooter />
    </>
  );
}
