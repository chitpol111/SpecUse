import { useState } from "react";

const links = [
  { label: "หน้าแรก", href: "#home" },
  { label: "จัดสเปก", href: "#start" },
  { label: "สเปกสาธารณะ", href: "#public-builds" },
];

export default function SiteHeader() {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="sticky top-0 z-50 border-b border-navy-900/5 bg-cream/90 backdrop-blur-xl">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-5 py-4 sm:px-8">
        <a
          href="#home"
          className="flex items-center gap-2.5"
          aria-label="SpecUse หน้าแรก"
          onClick={() => setMenuOpen(false)}
        >
          <span className="grid h-10 w-10 place-items-center rounded-xl bg-navy-900 text-white shadow-sm">
            <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" aria-hidden="true">
              <path d="M5 7.5h14M5 12h14M5 16.5h8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
              <circle cx="18" cy="16.5" r="2" fill="#16b8a6" />
            </svg>
          </span>
          <span className="text-xl font-extrabold tracking-tight text-navy-900">
            Spec<span className="text-aqua-600">Use</span>
          </span>
        </a>

        <button
          type="button"
          className="grid h-10 w-10 place-items-center rounded-xl border border-navy-900/10 text-navy-900 md:hidden"
          aria-label={menuOpen ? "ปิดเมนู" : "เปิดเมนู"}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((open) => !open)}
        >
          {menuOpen ? (
            <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" aria-hidden="true">
              <path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" aria-hidden="true">
              <path d="M4 7h16M4 12h16M4 17h16" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            </svg>
          )}
        </button>

        <nav className="hidden items-center gap-8 md:flex" aria-label="เมนูหลัก">
          {links.map((link) => (
            <a key={link.href} className="text-sm font-semibold text-slate-600 transition hover:text-aqua-600" href={link.href}>
              {link.label}
            </a>
          ))}
          <a
            href="#member"
            className="rounded-full border border-navy-900/15 px-5 py-2.5 text-sm font-bold text-navy-900 transition hover:border-aqua-500 hover:text-aqua-600"
          >
            เข้าสู่ระบบ
          </a>
        </nav>
      </div>

      {menuOpen && (
        <nav className="border-t border-navy-900/5 bg-cream px-5 py-4 md:hidden" aria-label="เมนูมือถือ">
          <div className="mx-auto flex max-w-7xl flex-col gap-1">
            {links.map((link) => (
              <a
                key={link.href}
                className="rounded-lg px-3 py-3 font-semibold text-slate-700 hover:bg-white"
                href={link.href}
                onClick={() => setMenuOpen(false)}
              >
                {link.label}
              </a>
            ))}
            <a
              href="#member"
              className="mt-1 rounded-lg bg-navy-900 px-3 py-3 font-bold text-white"
              onClick={() => setMenuOpen(false)}
            >
              เข้าสู่ระบบ
            </a>
          </div>
        </nav>
      )}
    </header>
  );
}
