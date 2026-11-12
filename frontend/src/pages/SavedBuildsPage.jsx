import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { validateBuild } from "../api/specUseApi";
import { useAuth } from "../auth/AuthContext";
import SiteFooter from "../components/SiteFooter";
import SiteHeader from "../components/SiteHeader";

const buttonClass = "min-h-11 rounded-xl border border-white/20 px-4 py-2 text-sm font-bold text-slate-100 hover:border-ruby-400 hover:text-ruby-200 disabled:opacity-50";
const inputClass = "mt-2 w-full rounded-xl border border-white/15 bg-obsidian-900 px-3 py-2 text-slate-100 focus:border-ruby-400 focus:outline-none focus:ring-2 focus:ring-ruby-400/30";
const formatCount = (value) => Number.isSafeInteger(value) ? String(value) : "เกินช่วงที่รองรับ";

function isSafeDraft(build) {
  return build.items.every((item) =>
    Number.isSafeInteger(item.quantity) && Number.isSafeInteger(item.owned_quantity)) &&
    build.ram_items.every((item) =>
      Number.isSafeInteger(item.required_modules) && Number.isSafeInteger(item.owned_modules));
}

export default function SavedBuildsPage() {
  const { authenticated, client } = useAuth();
  const [page, setPage] = useState(1);
  const [listing, setListing] = useState(null);
  const [listLoading, setListLoading] = useState(false);
  const [listError, setListError] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const [build, setBuild] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [price, setPrice] = useState(null);
  const [priceError, setPriceError] = useState("");
  const [pricePending, setPricePending] = useState(false);
  const [reloadList, setReloadList] = useState(0);
  const [reloadDetail, setReloadDetail] = useState(0);
  const requestGeneration = useRef(0);

  useEffect(() => {
    requestGeneration.current += 1;
    setPending(false);
    setPricePending(false);
    return () => { requestGeneration.current += 1; };
  }, [authenticated, selectedId]);

  useEffect(() => {
    if (!authenticated) {
      setListing(null);
      setPage(1);
      setSelectedId("");
      setBuild(null);
      setMessage("");
      setDetailError("");
      setPrice(null);
      setPriceError("");
      return undefined;
    }
    let active = true;
    setListLoading(true);
    setListError("");
    client.listBuilds(page)
      .then((data) => { if (active) setListing(data); })
      .catch((error) => { if (active) { setListing(null); setListError(error.message); } })
      .finally(() => { if (active) setListLoading(false); });
    return () => { active = false; };
  }, [authenticated, client, page, reloadList]);

  useEffect(() => {
    if (!authenticated || !selectedId) return undefined;
    let active = true;
    setBuild(null);
    setPrice(null);
    setPriceError("");
    setDetailError("");
    setDetailLoading(true);
    client.getBuild(selectedId)
      .then((data) => {
        if (!active) return;
        setBuild(data);
        setName(data.name);
        setDescription(data.description);
      })
      .catch((error) => { if (active) setDetailError(error.message); })
      .finally(() => { if (active) setDetailLoading(false); });
    return () => { active = false; };
  }, [authenticated, client, selectedId, reloadDetail]);

  const reload = useCallback(() => setReloadList((value) => value + 1), []);

  async function saveDetails(event) {
    event.preventDefault();
    if (pending || pricePending) return;
    const trimmed = name.trim();
    if (!trimmed || [...trimmed].length > 100 || [...description].length > 2000 ||
        trimmed.includes("\0") || description.includes("\0")) {
      setDetailError("ชื่อสเปกต้องมี 1–100 ตัวอักษร และคำอธิบายไม่เกิน 2,000 ตัวอักษร");
      return;
    }
    setPending(true);
    setDetailError("");
    setMessage("");
    const generation = requestGeneration.current;
    try {
      const updated = await client.updateBuildDetails(build.id, trimmed, description);
      if (requestGeneration.current !== generation) return;
      setBuild(updated);
      setName(updated.name);
      setDescription(updated.description);
      setPrice(null);
      setPriceError("");
      setMessage("บันทึกชื่อและคำอธิบายแล้ว");
      reload();
    } catch (error) {
      if (requestGeneration.current === generation) setDetailError(error.message);
    } finally {
      if (requestGeneration.current === generation) setPending(false);
    }
  }

  async function deleteBuild() {
    if (pending || pricePending) return;
    if (!window.confirm(`ลบสเปก "${build.name}" หรือไม่? การลบไม่สามารถย้อนกลับได้`)) return;
    setPending(true);
    setDetailError("");
    const generation = requestGeneration.current;
    try {
      await client.deleteBuild(build.id);
      if (requestGeneration.current !== generation) return;
      setSelectedId("");
      setBuild(null);
      setMessage("ลบสเปกแล้ว");
      if (page > 1 && listing?.items.length === 1) setPage(page - 1);
      else reload();
    } catch (error) {
      if (requestGeneration.current === generation) setDetailError(error.message);
    } finally {
      if (requestGeneration.current === generation) setPending(false);
    }
  }

  async function checkPrice() {
    if (pending || pricePending || !build.availability.ready_for_validation || !isSafeDraft(build)) return;
    setPrice(null);
    setPriceError("");
    setPricePending(true);
    const generation = requestGeneration.current;
    try {
      const data = await validateBuild({
        items: build.items.map(({ product_id, quantity, owned_quantity }) => ({ product_id, quantity, owned_quantity })),
        ram_items: build.ram_items.map(({ ram_model_id, required_modules, owned_modules }) => ({ ram_model_id, required_modules, owned_modules })),
      });
      if (!Number.isSafeInteger(data.total_price_thb) || data.total_price_thb < 0) {
        throw new Error("ผลราคาจาก Backend ไม่ตรงกับ API contract");
      }
      if (requestGeneration.current === generation) setPrice(data.total_price_thb);
    } catch (error) {
      if (requestGeneration.current === generation) setPriceError(error.message);
    } finally {
      if (requestGeneration.current === generation) setPricePending(false);
    }
  }

  return (
    <>
      <SiteHeader />
      <main className="min-h-[calc(100vh-5rem)] px-5 pb-20 pt-10 sm:px-8">
        <div className="mx-auto max-w-6xl">
          <div className="flex flex-wrap items-end justify-between gap-4 border-b border-white/10 pb-6">
            <div>
              <h1 className="text-3xl font-bold text-slate-100">สเปกของฉัน</h1>
              <p className="mt-2 text-sm leading-6 text-slate-300">สเปกที่บันทึกเป็นส่วนตัว ดูได้เฉพาะบัญชีของคุณ</p>
            </div>
            <Link to="/build" className={buttonClass}>จัดสเปกใหม่</Link>
          </div>
          {!authenticated ? (
            <div className="mt-8 rounded-2xl border border-white/10 bg-obsidian-900 p-6">
              <p className="text-slate-200">เข้าสู่ระบบเพื่อดูสเปกที่บันทึก</p>
              <Link to="/login" className="mt-4 inline-flex rounded-xl bg-ruby-500 px-5 py-3 font-bold text-white">เข้าสู่ระบบ</Link>
            </div>
          ) : (
            <div className="mt-8 grid items-start gap-6 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
              <section aria-labelledby="saved-list-heading" className="rounded-2xl border border-white/10 bg-obsidian-900 p-5 sm:p-6">
                <h2 id="saved-list-heading" className="text-xl font-bold">รายการสเปก</h2>
                {listLoading && <p role="status" className="mt-4 text-slate-300">กำลังโหลดสเปก...</p>}
                {listError && <p role="alert" className="mt-4 text-ruby-200">{listError} <button type="button" onClick={reload} className="underline">ลองอีกครั้ง</button></p>}
                {!listLoading && listing?.items.length === 0 && <p className="mt-4 text-sm text-slate-300">ยังไม่มีสเปกในหน้านี้ ไปจัดสเปกแล้วบันทึกได้เลย</p>}
                {!listLoading && listing?.items.length > 0 && (
                  <ul className="mt-4 divide-y divide-white/10">
                    {listing.items.map((item) => (
                      <li key={item.id}>
                        <button type="button" disabled={pending || pricePending} aria-pressed={selectedId === item.id}
                          onClick={() => { setMessage(""); setSelectedId(item.id); }}
                          className="w-full rounded-lg px-3 py-4 text-left hover:bg-white/5 disabled:opacity-50">
                          <span className="block break-words font-bold text-slate-100">{item.name}</span>
                          <span className="mt-1 block break-words text-sm text-slate-300">{item.description || "ไม่มีคำอธิบาย"}</span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
                {listing && (
                  <div className="mt-5 flex items-center justify-between gap-2 text-sm text-slate-300">
                    <button type="button" disabled={page <= 1 || listLoading || pending || pricePending} onClick={() => setPage(page - 1)} className={buttonClass}>ก่อนหน้า</button>
                    <span>หน้า {page}</span>
                    <button type="button" disabled={page * listing.page_size >= listing.total || listLoading || pending || pricePending} onClick={() => setPage(page + 1)} className={buttonClass}>ถัดไป</button>
                  </div>
                )}
                {message && !selectedId && <p role="status" className="mt-4 text-emerald-200">{message}</p>}
              </section>

              <section aria-labelledby="saved-detail-heading" className="rounded-2xl border border-white/10 bg-obsidian-900 p-5 sm:p-6">
                <h2 id="saved-detail-heading" className="text-xl font-bold">รายละเอียดสเปก</h2>
                {!selectedId && <p className="mt-4 text-sm text-slate-300">เลือกรายการเพื่อดูรายละเอียด</p>}
                {detailLoading && <p role="status" className="mt-4 text-slate-300">กำลังโหลดรายละเอียด...</p>}
                {detailError && <p role="alert" className="mt-4 text-ruby-200">{detailError} {!build && <button type="button" onClick={() => setReloadDetail((value) => value + 1)} className="underline">ลองอีกครั้ง</button>}</p>}
                {build && (
                  <>
                    <form onSubmit={saveDetails} className="mt-5 grid gap-4">
                      <label className="text-sm font-bold text-slate-200">ชื่อสเปก
                        <input value={name} onChange={(event) => { setName(event.target.value); setMessage(""); }} disabled={pending} required className={inputClass} />
                      </label>
                      <label className="text-sm font-bold text-slate-200">คำอธิบาย
                        <textarea value={description} onChange={(event) => { setDescription(event.target.value); setMessage(""); }} disabled={pending} rows={3} className={inputClass} />
                      </label>
                      <div className="flex flex-wrap gap-3">
                        <button type="submit" disabled={pending || pricePending} className="min-h-11 rounded-xl bg-ruby-500 px-5 py-2 text-sm font-bold text-white hover:bg-ruby-400 disabled:opacity-50">บันทึกข้อมูล</button>
                        <button type="button" disabled={pending || pricePending} onClick={deleteBuild} className={buttonClass}>ลบสเปก</button>
                      </div>
                    </form>
                    {message && <p role="status" className="mt-4 text-sm text-emerald-200">{message}</p>}
                    <h3 className="mt-7 border-t border-white/10 pt-5 font-bold">อุปกรณ์ที่บันทึก</h3>
                    <p className="mt-2 text-sm text-slate-300">รอบนี้แก้ไขได้เฉพาะชื่อและคำอธิบาย รายการอุปกรณ์เดิมจะถูกเก็บไว้ครบ</p>
                    {!isSafeDraft(build) && <p role="alert" className="mt-3 text-sm text-amber-200">จำนวนบางรายการเกินช่วงที่หน้าเว็บรองรับ จึงยังตรวจราคาไม่ได้</p>}
                    {build.items.length + build.ram_items.length === 0 && <p className="mt-4 text-sm text-slate-300">ยังไม่มีอุปกรณ์ในสเปกนี้</p>}
                    <ul className="mt-3 divide-y divide-white/10">
                      {build.items.map((item) => (
                        <li key={item.product_id} className="py-3 text-sm">
                          <p className="break-words font-bold">{item.product_summary.name} {!item.product_summary.is_active && <span className="text-amber-200">(ปิดใช้งาน)</span>}</p>
                          <p className="mt-1 text-slate-300">{item.product_summary.category} · ต้องการ {formatCount(item.quantity)} ชิ้น · มีแล้ว {formatCount(item.owned_quantity)} ชิ้น</p>
                        </li>
                      ))}
                      {build.ram_items.map((item) => (
                        <li key={item.ram_model_id} className="py-3 text-sm">
                          <p className="font-bold">RAM {item.ram_model_summary.ram_type}</p>
                          <p className="mt-1 break-all text-slate-300">รหัสรุ่น: {item.ram_model_id}</p>
                          <p className="mt-1 text-slate-300">ต้องการ {formatCount(item.required_modules)} โมดูล · มีแล้ว {formatCount(item.owned_modules)} โมดูล</p>
                        </li>
                      ))}
                    </ul>
                    {!build.availability.ready_for_validation && (
                      <div className="mt-4 rounded-lg border border-amber-300/20 p-3 text-sm text-amber-100">
                        <p>ยังตรวจราคาไม่ได้</p>
                        <ul className="mt-1 list-inside list-disc">{build.availability.issues.map((issue, index) => <li key={index}>{issue.message}</li>)}</ul>
                      </div>
                    )}
                    <button type="button" disabled={pending || pricePending || !build.availability.ready_for_validation || !isSafeDraft(build)}
                      onClick={checkPrice} className={`${buttonClass} mt-5`}>
                      {pricePending ? "กำลังตรวจราคา..." : "ตรวจราคาปัจจุบัน"}
                    </button>
                    {price !== null && <p role="status" className="mt-3 text-sm text-slate-200">ราคาอุปกรณ์ที่ยังไม่มี: {new Intl.NumberFormat("th-TH").format(price)} บาท</p>}
                    {priceError && <p role="alert" className="mt-3 text-sm text-ruby-200">ยังตรวจราคาไม่ได้: {priceError}</p>}
                  </>
                )}
              </section>
            </div>
          )}
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
