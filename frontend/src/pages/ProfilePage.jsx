import { useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import SiteFooter from "../components/SiteFooter";
import SiteHeader from "../components/SiteHeader";

const inputClass = "mt-2 min-h-12 w-full rounded-xl border border-white/15 bg-obsidian-900 px-4 py-3 text-base text-slate-100 outline-none transition hover:border-white/25 focus:border-ruby-400 focus:ring-2 focus:ring-ruby-400/30";
const primaryClass = "min-h-11 rounded-xl bg-ruby-500 px-5 py-3 text-sm font-bold text-white transition hover:bg-ruby-400 disabled:cursor-not-allowed disabled:opacity-50";
const secondaryClass = "min-h-11 rounded-xl border border-white/20 px-5 py-3 text-sm font-bold text-slate-100 transition hover:border-ruby-400 hover:text-ruby-200 disabled:opacity-50";

export default function ProfilePage() {
  const { authenticated, user, profile, client } = useAuth();
  const [editing, setEditing] = useState(false);
  const [profilePending, setProfilePending] = useState(false);
  const [profileError, setProfileError] = useState("");
  const [profileMessage, setProfileMessage] = useState("");
  const [passwordPending, setPasswordPending] = useState(false);
  const [passwordError, setPasswordError] = useState("");
  const [passwordMessage, setPasswordMessage] = useState("");

  async function handleProfileSave(event) {
    event.preventDefault();
    if (profilePending) return;
    const name = String(new FormData(event.currentTarget).get("displayName")).trim();
    setProfileMessage("");
    if (!name || [...name].length > 100 || name.includes("\0")) {
      setProfileError("ชื่อที่แสดงต้องมีความยาว 1–100 ตัวอักษร");
      return;
    }
    setProfilePending(true);
    setProfileError("");
    try {
      await client.updateProfile(name);
      setEditing(false);
      setProfileMessage("บันทึกชื่อที่แสดงแล้ว");
    } catch (error) {
      setProfileError(error.message);
    } finally {
      setProfilePending(false);
    }
  }

  async function handlePasswordSave(event) {
    event.preventDefault();
    if (passwordPending) return;
    const form = event.currentTarget;
    const values = new FormData(form);
    const currentPassword = String(values.get("currentPassword"));
    const newPassword = String(values.get("newPassword"));
    setPasswordMessage("");
    if (newPassword !== String(values.get("confirmPassword"))) {
      setPasswordError("รหัสผ่านใหม่และรหัสผ่านยืนยันไม่ตรงกัน");
      return;
    }
    setPasswordPending(true);
    setPasswordError("");
    try {
      await client.changePassword(currentPassword, newPassword);
      form.reset();
      setPasswordMessage("เปลี่ยนรหัสผ่านแล้ว");
    } catch (error) {
      setPasswordError(error.message);
    } finally {
      setPasswordPending(false);
    }
  }

  async function handleProfileRetry() {
    setProfilePending(true);
    setProfileError("");
    try {
      await client.loadProfile();
    } catch (error) {
      setProfileError(error.message);
    } finally {
      setProfilePending(false);
    }
  }

  return (
    <>
      <SiteHeader />
      <main className="min-h-[calc(100vh-5rem)] px-5 pb-20 pt-12 sm:px-8 sm:pt-16">
        <div className="mx-auto max-w-6xl">
          <div className="border-b border-white/10 pb-7">
            <h1 className="text-3xl font-bold text-slate-100">ข้อมูลส่วนตัว</h1>
            <p className="mt-2 text-sm leading-7 text-slate-300">จัดการข้อมูลบัญชีและรหัสผ่านของคุณ</p>
          </div>

          {!authenticated ? (
            <div className="mt-10 max-w-xl rounded-2xl border border-white/10 bg-obsidian-900 p-6">
              <h2 className="text-xl font-bold">เข้าสู่ระบบเพื่อดูข้อมูลส่วนตัว</h2>
              <p className="mt-2 text-sm leading-7 text-slate-300">ข้อมูลบัญชีจะแสดงหลังจากเข้าสู่ระบบ</p>
              {(profileError || passwordError) && (
                <p role="alert" className="mt-4 text-sm leading-6 text-ruby-200">{profileError || passwordError}</p>
              )}
              <Link to="/login" className="mt-5 inline-flex min-h-11 items-center rounded-xl bg-ruby-500 px-5 py-3 text-sm font-bold text-white hover:bg-ruby-400">เข้าสู่ระบบ</Link>
            </div>
          ) : !profile || !user ? (
            <div className="mt-10 space-y-4">
              <p role="status" className="text-slate-200">ยังโหลดข้อมูลสมาชิกไม่ได้</p>
              {profileError && <p role="alert" className="text-sm text-ruby-200">{profileError}</p>}
              <button type="button" disabled={profilePending} onClick={handleProfileRetry} className={secondaryClass}>
                {profilePending ? "กำลังโหลด..." : "ลองโหลดข้อมูลอีกครั้ง"}
              </button>
            </div>
          ) : (
            <div className="mt-8 grid items-start gap-8 lg:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
              <section aria-labelledby="account-heading" className="overflow-hidden rounded-2xl border border-white/10 bg-obsidian-900">
                <div className="flex flex-wrap items-start justify-between gap-4 border-b border-white/10 p-6 sm:p-8">
                  <div className="flex min-w-0 items-center gap-4">
                    <span className="grid h-14 w-14 shrink-0 place-items-center rounded-full border border-ruby-400/20 bg-ruby-500/10 text-ruby-200" aria-hidden="true">
                      <svg viewBox="0 0 24 24" fill="none" className="h-7 w-7" stroke="currentColor" strokeWidth="1.5">
                        <circle cx="12" cy="8" r="3.5" />
                        <path d="M5 20a7 7 0 0 1 14 0" strokeLinecap="round" />
                      </svg>
                    </span>
                    <div className="min-w-0">
                      <h2 id="account-heading" className="break-words text-xl font-bold">{profile.display_name}</h2>
                      <p className="mt-1 text-sm text-slate-300">{profile.role === "admin" ? "ผู้ดูแลระบบ" : "สมาชิก"}</p>
                    </div>
                  </div>
                  {!editing && (
                    <button type="button" onClick={() => { setEditing(true); setProfileMessage(""); setProfileError(""); }} className={secondaryClass}>
                      แก้ไขข้อมูลส่วนตัว
                    </button>
                  )}
                </div>

                <div className="p-6 sm:p-8">
                  {editing ? (
                    <form onSubmit={handleProfileSave} className="space-y-5">
                      <div>
                        <label htmlFor="profile-name" className="text-sm font-bold text-slate-200">ชื่อที่แสดง</label>
                        <input id="profile-name" name="displayName" type="text" defaultValue={profile.display_name} autoComplete="nickname" required className={inputClass} />
                        <p className="mt-2 text-xs leading-6 text-slate-300">ใช้ 1–100 ตัวอักษร</p>
                      </div>
                      {profileError && <p role="alert" className="text-sm leading-6 text-ruby-200">{profileError}</p>}
                      <div className="flex flex-wrap gap-3">
                        <button type="submit" disabled={profilePending} className={primaryClass}>{profilePending ? "กำลังบันทึก..." : "บันทึกข้อมูล"}</button>
                        <button type="button" disabled={profilePending} onClick={() => { setEditing(false); setProfileError(""); }} className={secondaryClass}>ยกเลิก</button>
                      </div>
                    </form>
                  ) : (
                    <dl className="grid gap-6 sm:grid-cols-2">
                      <div>
                        <dt className="text-sm text-slate-300">ชื่อที่แสดง</dt>
                        <dd className="mt-1 break-words font-bold text-slate-100">{profile.display_name}</dd>
                      </div>
                      <div>
                        <dt className="text-sm text-slate-300">อีเมล</dt>
                        <dd className="mt-1 break-all font-bold text-slate-100">{user.email}</dd>
                      </div>
                    </dl>
                  )}
                  {profileMessage && <p role="status" className="mt-5 text-sm text-slate-200">{profileMessage}</p>}
                </div>
              </section>

              <section aria-labelledby="security-heading" className="rounded-2xl border border-white/10 bg-obsidian-900 p-6 sm:p-8">
                <h2 id="security-heading" className="text-xl font-bold text-slate-100">เปลี่ยนรหัสผ่าน</h2>
                <p className="mt-2 text-sm leading-7 text-slate-300">กรอกรหัสผ่านปัจจุบันเพื่อยืนยันก่อนตั้งรหัสผ่านใหม่</p>
                <form onSubmit={handlePasswordSave} className="mt-6 space-y-5">
                  <div>
                    <label htmlFor="current-password" className="text-sm font-bold text-slate-200">รหัสผ่านปัจจุบัน</label>
                    <input id="current-password" name="currentPassword" type="password" autoComplete="current-password" required className={inputClass} />
                  </div>
                  <div>
                    <label htmlFor="new-password" className="text-sm font-bold text-slate-200">รหัสผ่านใหม่</label>
                    <input id="new-password" name="newPassword" type="password" autoComplete="new-password" minLength={8} required className={inputClass} />
                    <p className="mt-2 text-xs leading-6 text-slate-300">อย่างน้อย 8 ตัวอักษร</p>
                  </div>
                  <div>
                    <label htmlFor="confirm-password" className="text-sm font-bold text-slate-200">ยืนยันรหัสผ่านใหม่</label>
                    <input id="confirm-password" name="confirmPassword" type="password" autoComplete="new-password" minLength={8} required className={inputClass} />
                  </div>
                  {passwordError && <p role="alert" className="text-sm leading-6 text-ruby-200">{passwordError}</p>}
                  {passwordMessage && <p role="status" className="text-sm text-slate-200">{passwordMessage}</p>}
                  <button type="submit" disabled={passwordPending} className={`${primaryClass} w-full`}>
                    {passwordPending ? "กำลังเปลี่ยนรหัสผ่าน..." : "เปลี่ยนรหัสผ่าน"}
                  </button>
                </form>
              </section>
            </div>
          )}
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
