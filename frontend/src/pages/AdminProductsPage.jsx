import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import SiteFooter from "../components/SiteFooter";
import SiteHeader from "../components/SiteHeader";

const categories = [
  ["cpu", "CPU"],
  ["motherboard", "เมนบอร์ด"],
  ["ram", "RAM"],
  ["gpu", "การ์ดจอ"],
  ["storage", "อุปกรณ์จัดเก็บข้อมูล"],
  ["psu", "Power Supply"],
  ["case", "เคส"],
];
const inputClass = "mt-2 min-h-11 w-full rounded-xl border border-white/15 bg-obsidian-900 px-3 py-2.5 text-sm text-slate-100 outline-none transition placeholder:text-slate-400 hover:border-white/25 focus:border-ruby-400 focus:ring-2 focus:ring-ruby-400/30";
const primaryClass = "min-h-11 rounded-xl bg-ruby-500 px-4 py-2.5 text-sm font-bold text-white transition hover:bg-ruby-400 disabled:cursor-not-allowed disabled:opacity-50";
const secondaryClass = "min-h-10 rounded-xl border border-white/20 px-3 py-2 text-sm font-bold text-slate-100 transition hover:border-ruby-400 hover:text-ruby-200 disabled:opacity-50";
const emptyForm = {
  name: "",
  category: "cpu",
  brand: "",
  price: "",
  isActive: true,
  ramModelId: "",
  cpuSocket: "",
  ramType: "",
  modulesPerPackage: "",
  chipsetName: "",
  vramGB: "",
  storageType: "",
  storageGB: "",
  specsByCategory: {},
};

function categoryName(category) {
  return categories.find(([value]) => value === category)?.[1] || category;
}

function formatPrice(price) {
  return new Intl.NumberFormat("th-TH").format(price);
}

function capacityBytes(value) {
  if (!value) return null;
  if (!/^\d+(?:\.\d{1,9})?$/.test(value)) {
    throw new Error("ความจุ GB ต้องเป็นเลขทศนิยมไม่เกิน 9 ตำแหน่ง");
  }
  const [whole, fraction = ""] = value.split(".");
  const bytes = Number(whole + fraction.padEnd(9, "0"));
  if (!Number.isSafeInteger(bytes) || bytes <= 0) {
    throw new Error("ความจุต้องมากกว่า 0 และไม่เกินขนาดที่รองรับ");
  }
  return bytes;
}

function formFromProduct(product) {
  const specs = product.specs;
  return {
    name: product.name,
    category: product.category,
    brand: product.brand,
    price: String(product.price_thb),
    isActive: product.is_active,
    ramModelId: product.ram_model_id || "",
    cpuSocket: specs?.cpu_socket || "",
    ramType: specs?.ram_type || "",
    modulesPerPackage: specs?.modules_per_package ? String(specs.modules_per_package) : "",
    chipsetName: specs?.display?.chipset_name || "",
    vramGB: specs?.display?.vram_capacity_bytes != null ? String(specs.display.vram_capacity_bytes / 1e9) : "",
    storageType: specs?.display?.media_type || "",
    storageGB: specs?.display?.capacity_bytes != null ? String(specs.display.capacity_bytes / 1e9) : "",
    specsByCategory: { [product.category]: specs ? JSON.stringify(specs, null, 2) : "" },
  };
}

export default function AdminProductsPage() {
  const { authenticated, profile, client } = useAuth();
  const [products, setProducts] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageCount, setPageCount] = useState(1);
  const [category, setCategory] = useState("");
  const [search, setSearch] = useState("");
  const [searchDraft, setSearchDraft] = useState("");
  const [loading, setLoading] = useState(false);
  const [pending, setPending] = useState(false);
  const [checkingProfile, setCheckingProfile] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [selected, setSelected] = useState(null);
  const [form, setForm] = useState(emptyForm);

  const loadProducts = useCallback(async () => {
    if (!authenticated || profile?.role !== "admin") return;
    setLoading(true);
    setError("");
    try {
      const result = await client.fetchAdminProducts({
        category,
        search,
        page,
        pageSize: 20,
      });
      const nextPageCount = Math.max(1, Math.ceil(result.total / result.page_size));
      setProducts(result.items);
      setTotal(result.total);
      setPageCount(nextPageCount);
      if (page > nextPageCount) setPage(nextPageCount);
    } catch (requestError) {
      setError(requestError.status === 403
        ? "บัญชีนี้ไม่มีสิทธิ์จัดการสินค้า"
        : requestError.message);
    } finally {
      setLoading(false);
    }
  }, [authenticated, profile?.role, client, category, search, page]);

  useEffect(() => {
    loadProducts();
  }, [loadProducts]);

  function startCreate() {
    setSelected(null);
    setForm(emptyForm);
    setError("");
    setMessage("");
  }

  function startEdit(product) {
    setSelected(product);
    setForm(formFromProduct(product));
    setError("");
    setMessage("");
  }

  function updateForm(event) {
    const { name, value, type, checked } = event.target;
    setForm((current) => ({ ...current, [name]: type === "checkbox" ? checked : value }));
  }

  function makePayload() {
    const name = form.name.trim();
    const brand = form.brand.trim();
    const price = Number(form.price);
    if (!name || !brand || form.price.trim() === "" || !Number.isSafeInteger(price) || price < 0) {
      throw new Error("กรอกชื่อสินค้า แบรนด์ และราคาเป็นจำนวนเงินบาทตั้งแต่ 0 ขึ้นไป");
    }

    let specs;
    const specsText = form.specsByCategory[form.category] || "";
    if (specsText.trim()) {
      try {
        specs = JSON.parse(specsText);
      } catch {
        throw new Error("ข้อมูลสเปกต้องเป็น JSON ที่ถูกต้อง");
      }
      if (!specs || Array.isArray(specs) || typeof specs !== "object") {
        throw new Error("ข้อมูลสเปกต้องเป็น JSON object");
      }
    }

    if (form.category === "cpu" || form.category === "motherboard") {
      if (!form.cpuSocket.trim()) throw new Error("กรุณาระบุซ็อกเก็ต CPU");
      specs = { ...specs, cpu_socket: form.cpuSocket.trim() };
    }
    if (form.category === "motherboard" || form.category === "ram") {
      if (!form.ramType.trim()) throw new Error("กรุณาระบุชนิด RAM");
      specs = { ...specs, ram_type: form.ramType.trim() };
    }
    if (form.category === "ram") {
      if (!form.ramModelId.trim()) throw new Error("RAM ต้องระบุ ram_model_id ที่มีอยู่ในฐานข้อมูล");
      if (!["1", "2"].includes(form.modulesPerPackage)) {
        throw new Error("กรุณาเลือกจำนวนแรมต่อแพ็ก");
      }
      specs = { ...specs, modules_per_package: Number(form.modulesPerPackage) };
    }

    if (form.category === "gpu" || form.category === "storage") {
      const isGpu = form.category === "gpu";
      const originalDisplay = selected?.specs?.display;
      const label = isGpu ? form.chipsetName.trim() : form.storageType;
      const capacity = isGpu ? form.vramGB.trim() : form.storageGB.trim();
      const originalLabel = isGpu ? originalDisplay?.chipset_name || "" : originalDisplay?.media_type || "";
      const originalCapacity = isGpu ? originalDisplay?.vram_capacity_bytes : originalDisplay?.capacity_bytes;
      const originalCapacityText = originalCapacity == null ? "" : String(originalCapacity / 1e9);
      if (label !== originalLabel || capacity !== originalCapacityText) {
        const bytes = capacityBytes(capacity);
        const display = { ...specs?.display };
        if (isGpu) {
          display.chipset_name = label || null;
          display.vram_capacity_bytes = bytes;
        } else {
          display.media_type = label || null;
          display.capacity_bytes = bytes;
        }
        specs = { ...specs, display };
      }
    }

    const payload = {
      name,
      category: form.category,
      brand,
      price_thb: price,
      is_active: form.isActive,
    };
    if (specs) payload.specs = specs;
    if (form.category === "ram") payload.ram_model_id = form.ramModelId.trim();
    return payload;
  }

  async function handleSave(event) {
    event.preventDefault();
    if (pending) return;
    setPending(true);
    setError("");
    setMessage("");
    try {
      const payload = makePayload();
      if (selected) {
        const { category: _category, ...editable } = payload;
        const result = await client.updateAdminProduct(selected.id, editable);
        setSelected(result.product);
        setForm(formFromProduct(result.product));
        setMessage(`บันทึกการแก้ไข ${result.product.name} แล้ว`);
      } else {
        const result = await client.createAdminProduct(payload);
        setMessage(`เพิ่มสินค้า ${result.product.name} แล้ว`);
      }
      await loadProducts();
      if (!selected) {
        setSelected(null);
        setForm(emptyForm);
      }
    } catch (saveError) {
      setError(saveError.status === 403
        ? "บัญชีนี้ไม่มีสิทธิ์จัดการสินค้า"
        : saveError.message);
    } finally {
      setPending(false);
    }
  }

  async function handleDeactivate(product) {
    if (!window.confirm(`ปิดใช้งาน “${product.name}” หรือไม่? สินค้าจะยังอยู่ในประวัติและสามารถเปิดใช้งานใหม่ได้`)) return;
    setPending(true);
    setError("");
    setMessage("");
    try {
      await client.deactivateAdminProduct(product.id);
      setMessage(`ปิดใช้งาน ${product.name} แล้ว`);
      if (selected?.id === product.id) {
        setForm((current) => ({ ...current, isActive: false }));
      }
      await loadProducts();
    } catch (requestError) {
      setError(requestError.status === 403
        ? "บัญชีนี้ไม่มีสิทธิ์จัดการสินค้า"
        : requestError.message);
    } finally {
      setPending(false);
    }
  }

  if (!authenticated) {
    return (
      <>
        <SiteHeader />
        <main className="min-h-[calc(100vh-5rem)] px-5 py-16 sm:px-8">
          <div className="mx-auto max-w-xl rounded-2xl border border-white/10 bg-obsidian-900 p-6 sm:p-8">
            <h1 className="text-3xl font-bold">จัดการสินค้า</h1>
            <p className="mt-3 text-sm leading-7 text-slate-300">เข้าสู่ระบบด้วยบัญชีผู้ดูแลเพื่อจัดการแค็ตตาล็อก</p>
            <Link to="/login" className={`${primaryClass} mt-5 inline-flex items-center`}>เข้าสู่ระบบ</Link>
          </div>
        </main>
        <SiteFooter />
      </>
    );
  }

  if (!profile || profile.role !== "admin") {
    return (
      <>
        <SiteHeader />
        <main className="min-h-[calc(100vh-5rem)] px-5 py-16 sm:px-8">
          <div className="mx-auto max-w-xl">
            <h1 className="text-3xl font-bold">จัดการสินค้า</h1>
            <p role="alert" className="mt-4 text-sm leading-7 text-ruby-200">
              {!profile ? "ยังไม่ได้โหลดสิทธิ์ผู้ดูแล" : "บัญชีนี้ไม่มีสิทธิ์จัดการสินค้า"}
            </p>
            {!profile && <button type="button" disabled={checkingProfile} onClick={async () => {
              setCheckingProfile(true);
              setError("");
              try {
                await client.loadProfile();
              } catch (requestError) {
                setError(requestError.message);
              } finally {
                setCheckingProfile(false);
              }
            }} className={`${secondaryClass} mt-4`}>
              {checkingProfile ? "กำลังตรวจสอบ..." : "ลองตรวจสอบอีกครั้ง"}
            </button>}
          </div>
        </main>
        <SiteFooter />
      </>
    );
  }

  return (
    <>
      <SiteHeader />
      <main className="min-h-[calc(100vh-5rem)] px-5 pb-20 pt-10 sm:px-8 sm:pt-14">
        <div className="mx-auto max-w-7xl">
          <div className="flex flex-wrap items-end justify-between gap-5 border-b border-white/10 pb-6">
            <div>
              <h1 className="text-3xl font-bold text-slate-100">จัดการสินค้า</h1>
              <p className="mt-2 text-sm leading-7 text-slate-300">เพิ่ม แก้ไขราคา และปิดใช้งานสินค้าในแค็ตตาล็อก</p>
            </div>
            <button type="button" onClick={startCreate} className={primaryClass}>เพิ่มสินค้า</button>
          </div>

          {(error || message) && (
            <p role={error ? "alert" : "status"} className={`mt-5 rounded-xl border px-4 py-3 text-sm leading-6 ${error ? "border-ruby-400/30 bg-ruby-950/30 text-ruby-200" : "border-white/10 bg-obsidian-900 text-slate-200"}`}>
              {error || message}
            </p>
          )}

          <div className="mt-7 grid items-start gap-8 xl:grid-cols-[minmax(0,1.3fr)_minmax(20rem,0.7fr)]">
            <section aria-labelledby="product-list-heading" className="min-w-0">
              <div className="flex flex-wrap items-end justify-between gap-4">
                <h2 id="product-list-heading" className="text-xl font-bold">รายการสินค้า <span className="text-sm font-normal text-slate-300">({total})</span></h2>
                <form onSubmit={(event) => { event.preventDefault(); setPage(1); setSearch(searchDraft.trim()); }} className="grid w-full gap-3 sm:grid-cols-[minmax(0,1fr)_12rem_auto] xl:max-w-2xl">
                  <label className="sr-only" htmlFor="product-search">ค้นหาชื่อสินค้า</label>
                  <input id="product-search" value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="ค้นหาชื่อสินค้า" className={inputClass} />
                  <label className="sr-only" htmlFor="product-category-filter">กรองหมวดหมู่</label>
                  <select id="product-category-filter" value={category} onChange={(event) => { setCategory(event.target.value); setPage(1); }} className={inputClass}>
                    <option value="">ทุกหมวดหมู่</option>
                    {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                  </select>
                  <button type="submit" className={secondaryClass}>ค้นหา</button>
                </form>
              </div>

              {loading ? (
                <div aria-label="กำลังโหลดรายการสินค้า" className="mt-5 space-y-3">
                  {[0, 1, 2].map((row) => <div key={row} className="h-20 animate-pulse rounded-xl border border-white/10 bg-obsidian-900" />)}
                </div>
              ) : products.length === 0 ? (
                <div className="mt-5 rounded-2xl border border-white/10 bg-obsidian-900 px-5 py-8">
                  <h3 className="font-bold">ไม่พบสินค้า</h3>
                  <p className="mt-2 text-sm leading-6 text-slate-300">ลองเปลี่ยนคำค้นหาหรือหมวดหมู่ หรือเพิ่มสินค้าใหม่</p>
                </div>
              ) : (
                <>
                  <div className="mt-5 hidden overflow-x-auto rounded-2xl border border-white/10 lg:block">
                    <table className="w-full min-w-[42rem] text-left text-sm">
                      <thead className="bg-obsidian-900 text-slate-300">
                        <tr>
                          <th scope="col" className="px-4 py-3 font-semibold">สินค้า</th>
                          <th scope="col" className="px-4 py-3 font-semibold">หมวดหมู่</th>
                          <th scope="col" className="px-4 py-3 text-right font-semibold">ราคา</th>
                          <th scope="col" className="px-4 py-3 font-semibold">สถานะ</th>
                          <th scope="col" className="px-4 py-3"><span className="sr-only">การทำงาน</span></th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-white/10">
                        {products.map((product) => (
                          <tr key={product.id} className={selected?.id === product.id ? "bg-ruby-500/10" : "bg-obsidian-950"}>
                            <td className="px-4 py-4">
                              <p className="font-bold text-slate-100">{product.name}</p>
                              <p className="mt-1 text-xs text-slate-300">{product.brand}</p>
                            </td>
                            <td className="px-4 py-4 text-slate-200">{categoryName(product.category)}</td>
                            <td className="px-4 py-4 text-right tabular-nums text-slate-100">{formatPrice(product.price_thb)} ฿</td>
                            <td className="px-4 py-4">
                              <span className={product.is_active ? "text-emerald-200" : "text-slate-300"}>{product.is_active ? "ใช้งาน" : "ปิดใช้งาน"}</span>
                            </td>
                            <td className="px-4 py-4 text-right">
                              <button type="button" onClick={() => startEdit(product)} className={secondaryClass}>แก้ไข</button>
                              {product.is_active && <button type="button" onClick={() => handleDeactivate(product)} disabled={pending} className="ml-2 min-h-10 rounded-xl px-3 py-2 text-sm font-bold text-ruby-200 hover:bg-ruby-500/10 disabled:opacity-50">ปิดใช้งาน</button>}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  <ul className="mt-5 space-y-3 lg:hidden">
                    {products.map((product) => (
                      <li key={product.id} className={`rounded-2xl border p-4 ${selected?.id === product.id ? "border-ruby-400/40 bg-ruby-500/10" : "border-white/10 bg-obsidian-900"}`}>
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0">
                            <h3 className="break-words font-bold text-slate-100">{product.name}</h3>
                            <p className="mt-1 text-sm text-slate-300">{product.brand} · {categoryName(product.category)}</p>
                          </div>
                          <span className={product.is_active ? "shrink-0 text-xs text-emerald-200" : "shrink-0 text-xs text-slate-300"}>{product.is_active ? "ใช้งาน" : "ปิดใช้งาน"}</span>
                        </div>
                        <p className="mt-3 tabular-nums text-sm text-slate-100">{formatPrice(product.price_thb)} ฿</p>
                        <div className="mt-4 flex flex-wrap gap-2">
                          <button type="button" onClick={() => startEdit(product)} className={secondaryClass}>แก้ไขสินค้า</button>
                          {product.is_active && <button type="button" onClick={() => handleDeactivate(product)} disabled={pending} className="min-h-10 rounded-xl px-3 py-2 text-sm font-bold text-ruby-200 hover:bg-ruby-500/10 disabled:opacity-50">ปิดใช้งาน</button>}
                        </div>
                      </li>
                    ))}
                  </ul>
                </>
              )}

              <div className="mt-5 flex items-center justify-between gap-4">
                <p className="text-sm text-slate-300">หน้า {page} จาก {pageCount}</p>
                <div className="flex gap-2">
                  <button type="button" disabled={page <= 1 || loading} onClick={() => setPage((current) => current - 1)} className={secondaryClass}>ก่อนหน้า</button>
                  <button type="button" disabled={page >= pageCount || loading} onClick={() => setPage((current) => current + 1)} className={secondaryClass}>ถัดไป</button>
                </div>
              </div>
            </section>

            <section aria-labelledby="product-form-heading" className="rounded-2xl border border-white/10 bg-obsidian-900 p-5 sm:p-7">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 id="product-form-heading" className="text-xl font-bold">{selected ? "แก้ไขสินค้า" : "เพิ่มสินค้า"}</h2>
                  <p className="mt-2 text-sm leading-6 text-slate-300">{selected ? "แก้ไขข้อมูลที่อนุญาต หมวดหมู่เปลี่ยนไม่ได้" : "กรอกข้อมูลสินค้าและสเปกที่ตรวจสอบแล้ว"}</p>
                </div>
                {selected && <button type="button" onClick={startCreate} className="text-sm font-bold text-ruby-200 underline underline-offset-4">รายการใหม่</button>}
              </div>

              <form onSubmit={handleSave} className="mt-6 space-y-4">
                <div>
                  <label htmlFor="product-name" className="text-sm font-bold text-slate-200">ชื่อสินค้า</label>
                  <input id="product-name" name="name" value={form.name} onChange={updateForm} required maxLength={200} className={inputClass} />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div>
                    <label htmlFor="product-category" className="text-sm font-bold text-slate-200">หมวดหมู่</label>
                    <select id="product-category" name="category" value={form.category} onChange={updateForm} disabled={Boolean(selected)} className={inputClass}>
                      {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                    </select>
                  </div>
                  <div>
                    <label htmlFor="product-brand" className="text-sm font-bold text-slate-200">แบรนด์</label>
                    <input id="product-brand" name="brand" value={form.brand} onChange={updateForm} required maxLength={100} className={inputClass} />
                  </div>
                </div>
                <div>
                  <label htmlFor="product-price" className="text-sm font-bold text-slate-200">ราคาอ้างอิง (บาท)</label>
                  <input id="product-price" name="price" type="number" min="0" step="1" value={form.price} onChange={updateForm} required className={inputClass} />
                </div>
                <fieldset className="space-y-4 border-t border-white/10 pt-4">
                  <legend className="px-1 text-sm font-bold text-slate-100">ข้อมูลสเปก</legend>
                  {(form.category === "cpu" || form.category === "motherboard") && (
                    <div>
                      <label htmlFor="cpu-socket" className="text-sm font-bold text-slate-200">ซ็อกเก็ต CPU</label>
                      <input id="cpu-socket" name="cpuSocket" value={form.cpuSocket} onChange={updateForm} required placeholder="เช่น AM5" className={inputClass} />
                      <p className="mt-2 text-xs leading-6 text-slate-300">ใช้ตรวจว่าซ็อกเก็ต CPU ตรงกับเมนบอร์ด</p>
                    </div>
                  )}
                  {(form.category === "motherboard" || form.category === "ram") && (
                    <div>
                      <label htmlFor="ram-type" className="text-sm font-bold text-slate-200">ชนิด RAM</label>
                      <input id="ram-type" name="ramType" value={form.ramType} onChange={updateForm} required placeholder="เช่น DDR5" className={inputClass} />
                      <p className="mt-2 text-xs leading-6 text-slate-300">ระบุชนิดให้ตรงกับข้อมูลสินค้า เช่น DDR4 หรือ DDR5</p>
                    </div>
                  )}
                  {form.category === "ram" && (
                    <div>
                      <label htmlFor="ram-modules" className="text-sm font-bold text-slate-200">จำนวนแรมต่อแพ็ก</label>
                      <select id="ram-modules" name="modulesPerPackage" value={form.modulesPerPackage} onChange={updateForm} required className={inputClass}>
                        <option value="">เลือกจำนวน</option>
                        <option value="1">1 แถว</option>
                        <option value="2">2 แถว</option>
                      </select>
                    </div>
                  )}
                  {form.category === "gpu" && (
                    <>
                      <div>
                        <label htmlFor="gpu-chipset" className="text-sm font-bold text-slate-200">ชื่อชิปการ์ดจอ <span className="font-normal text-slate-300">(ถ้าทราบ)</span></label>
                        <input id="gpu-chipset" name="chipsetName" value={form.chipsetName} onChange={updateForm} className={inputClass} />
                      </div>
                      <div>
                        <label htmlFor="gpu-vram" className="text-sm font-bold text-slate-200">หน่วยความจำการ์ดจอ (GB) <span className="font-normal text-slate-300">(ถ้าทราบ)</span></label>
                        <input id="gpu-vram" name="vramGB" type="number" min="0" step="any" value={form.vramGB} onChange={updateForm} className={inputClass} />
                      </div>
                    </>
                  )}
                  {form.category === "storage" && (
                    <>
                      <div>
                        <label htmlFor="storage-type" className="text-sm font-bold text-slate-200">ชนิดอุปกรณ์จัดเก็บ <span className="font-normal text-slate-300">(ถ้าทราบ)</span></label>
                        <select id="storage-type" name="storageType" value={form.storageType} onChange={updateForm} className={inputClass}>
                          <option value="">ไม่ระบุ</option>
                          <option value="ssd">SSD</option>
                          <option value="hdd">HDD</option>
                          <option value="other">อื่น ๆ</option>
                        </select>
                      </div>
                      <div>
                        <label htmlFor="storage-capacity" className="text-sm font-bold text-slate-200">ความจุ (GB) <span className="font-normal text-slate-300">(ถ้าทราบ)</span></label>
                        <input id="storage-capacity" name="storageGB" type="number" min="0" step="any" value={form.storageGB} onChange={updateForm} className={inputClass} />
                      </div>
                    </>
                  )}
                  {(form.category === "gpu" || form.category === "storage") && (
                    <p className="text-xs leading-6 text-slate-300">ระบุเฉพาะข้อมูลที่ตรวจสอบแล้ว ไม่ทราบให้เว้นว่าง ความจุ 1 GB เท่ากับ 1,000,000,000 ไบต์</p>
                  )}
                  {(form.category === "psu" || form.category === "case") && (
                    <p className="text-sm leading-6 text-slate-300">หมวดนี้ยังไม่มีช่องสเปกที่จำเป็นต่อการบันทึก</p>
                  )}
                </fieldset>
                {form.category === "ram" && (
                  <div>
                    <label htmlFor="ram-model-id" className="text-sm font-bold text-slate-200">รหัสรุ่น RAM (RAM model ID)</label>
                    <input id="ram-model-id" name="ramModelId" value={form.ramModelId} onChange={updateForm} required className={inputClass} />
                    <p className="mt-2 text-xs leading-6 text-slate-300">ต้องใช้รหัสรุ่นที่เตรียมไว้ในฐานข้อมูลแล้ว รุ่นเดียวกันต้องมีชนิด RAM ตรงกัน</p>
                  </div>
                )}
                <details className="rounded-xl border border-white/10 px-4 py-3">
                  <summary className="cursor-pointer text-sm font-bold text-slate-200">ข้อมูลสเปกขั้นสูง (JSON)</summary>
                  <p className="mt-3 text-xs leading-6 text-slate-300">สำหรับข้อมูลเพิ่มเติมตามสัญญา Backend เท่านั้น หากแก้ช่องสเปกด้านบน ค่าจากช่องนั้นจะใช้แทนค่าที่ซ้ำกัน</p>
                  <label htmlFor="product-specs" className="mt-3 block text-sm text-slate-200">ข้อมูลสเปกขั้นสูง</label>
                  <textarea id="product-specs" value={form.specsByCategory[form.category] || ""} onChange={(event) => {
                    const value = event.target.value;
                    setForm((current) => ({ ...current, specsByCategory: { ...current.specsByCategory, [current.category]: value } }));
                  }} rows={7} spellCheck="false" className={`${inputClass} min-h-40 font-mono`} />
                </details>
                <label className="flex min-h-11 items-center gap-3 text-sm text-slate-200">
                  <input type="checkbox" name="isActive" checked={form.isActive} onChange={updateForm} className="h-4 w-4 accent-ruby-500" />
                  เปิดแสดงในแค็ตตาล็อก
                </label>
                <p className="text-xs leading-6 text-slate-300">การปิดใช้งานจะเก็บแถวสินค้าไว้เพื่อรักษารายการอ้างอิง</p>
                <button type="submit" disabled={pending || loading} className={`${primaryClass} w-full`}>
                  {pending ? "กำลังบันทึก..." : selected ? "บันทึกการแก้ไข" : "เพิ่มสินค้า"}
                </button>
              </form>
            </section>
          </div>
        </div>
      </main>
      <SiteFooter />

    </>
  );
}
