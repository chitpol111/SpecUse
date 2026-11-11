import { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import SiteFooter from "../components/SiteFooter";
import SiteHeader from "../components/SiteHeader";

const inputClassName =
  "mt-2 min-h-12 w-full rounded-xl border border-white/15 bg-obsidian-900 px-4 py-3 text-base text-slate-100 outline-none transition placeholder:text-slate-400 hover:border-white/25 focus:border-ruby-400 focus:ring-2 focus:ring-ruby-400/30";

export default function AuthPage({ mode }) {
  const { authenticated, profile, client } = useAuth();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const isRegister = mode === "register";
  const heading = isRegister ? "สมัครสมาชิก" : "เข้าสู่ระบบ";

  async function loadProfile() {
    try {
      await client.loadProfile();
      setError("");
    } catch (err) {
      setError(`โหลดข้อมูลสมาชิกไม่สำเร็จ: ${err.message}`);
    }
  }

  async function handleSubmit(event) {
    event.preventDefault();
    if (pending) return;
    const form = new FormData(event.currentTarget);
    const email = String(form.get("email")).trim();
    const password = String(form.get("password"));
    const displayName = isRegister ? String(form.get("displayName")).trim() : "";
    if (isRegister && (!displayName || [...displayName].length > 100 || displayName.includes("\0"))) {
      setError("ชื่อที่แสดงต้องมีความยาว 1–100 ตัวอักษร");
      return;
    }
    setPending(true);
    setError("");
    try {
      if (isRegister) {
        await client.register({ email, password, display_name: displayName });
      } else {
        await client.login({ email, password });
      }
      await loadProfile();
    } catch (err) {
      setError(err.message);
    } finally {
      setPending(false);
    }
  }

  async function handleProfileRetry() {
    setPending(true);
    await loadProfile();
    setPending(false);
  }

  return (
    <>
      <SiteHeader />
      <main className="relative min-h-[calc(100vh-5rem)] px-5 pb-20 pt-16 sm:px-8 sm:pt-24">
        <div className="relative mx-auto w-full max-w-md">
          <h1 className="text-3xl font-bold text-slate-100">{heading}</h1>
          <p className="mt-3 text-sm leading-7 text-slate-300">
            {isRegister ? "สร้างบัญชีเพื่อใช้งาน SpecUse" : "เข้าสู่บัญชี SpecUse ของคุณ"}
          </p>

          {authenticated ? (
            <div className="mt-8 space-y-5">
              {profile ? (
                <>
                  <p role="status">เข้าสู่ระบบแล้ว: {profile.display_name} ({profile.role})</p>
                  <p className="text-sm text-slate-300">เมื่อรีเฟรชหรือปิดหน้าเว็บ จะต้องเข้าสู่ระบบใหม่</p>
                  <Link to="/profile" className="block font-bold text-ruby-300 underline">ดูข้อมูลส่วนตัว</Link>
                  <Link to="/build" className="font-bold text-ruby-300 underline">ไปหน้าจัดสเปก</Link>
                </>
              ) : (
                <>
                  <p role="status">เข้าสู่ระบบแล้ว แต่ยังไม่ได้รับข้อมูลสมาชิก</p>
                  <button type="button" disabled={pending} onClick={handleProfileRetry} className="font-bold text-ruby-300 underline disabled:opacity-50">
                    ลองโหลดข้อมูลสมาชิกอีกครั้ง
                  </button>
                </>
              )}
              {error && <p role="alert" className="text-sm leading-6 text-ruby-200">{error}</p>}
            </div>
          ) : (
          <>
          <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
            {isRegister && (
              <div>
                <label htmlFor="display-name" className="text-sm font-bold text-slate-200">
                  ชื่อที่แสดง
                </label>
                <input
                  id="display-name"
                  name="displayName"
                  type="text"
                  autoComplete="nickname"
                  placeholder="ชื่อที่แสดง"
                  required
                  className={inputClassName}
                />
              </div>
            )}
            <div>
              <label htmlFor="email" className="text-sm font-bold text-slate-200">
                อีเมล
              </label>
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                placeholder="คุณ@example.com"
                required
                className={inputClassName}
              />
            </div>
            <div>
              <label htmlFor="password" className="text-sm font-bold text-slate-200">
                รหัสผ่าน
              </label>
              <input
                id="password"
                name="password"
                type="password"
                autoComplete={isRegister ? "new-password" : "current-password"}
                placeholder="รหัสผ่าน"
                minLength={isRegister ? 8 : undefined}
                required
                className={inputClassName}
              />
            </div>
            <button
              type="submit"
              disabled={pending}
              className="min-h-12 w-full rounded-xl bg-ruby-500 px-5 py-3 text-sm font-bold text-white transition hover:bg-ruby-400 active:bg-ruby-600 disabled:opacity-50"
            >
              {pending ? "กำลังดำเนินการ..." : heading}
            </button>
            {error && (
              <p role="alert" className="rounded-xl border border-ruby-400/25 bg-obsidian-800 px-4 py-3 text-sm leading-6 text-slate-200">
                {error}
              </p>
            )}
          </form>

          <p className="mt-8 text-sm text-slate-300">
            {isRegister ? "มีบัญชีแล้ว?" : "ยังไม่มีบัญชี?"}{" "}
            <Link
              to={isRegister ? "/login" : "/register"}
              className="font-bold text-ruby-300 underline-offset-4 hover:text-ruby-200 hover:underline"
            >
              {isRegister ? "เข้าสู่ระบบ" : "สมัครสมาชิก"}
            </Link>
          </p>
          </>
          )}
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
