import { useState } from "react";
import { Link, useLocation } from "react-router-dom";

const sectionLinks = [
  { label: "หน้าแรก", href: "#home" },
  { label: "วิธีใช้งาน", href: "#start" },
  { label: "ตัวอย่างเส้นทาง", href: "#public-builds" },
];

export default function SiteHeader() {
  const [menuOpen, setMenuOpen] = useState(false);
  const { pathname } = useLocation();
  const isHome = pathname === "/";
  const links = sectionLinks.map((link) => ({
    ...link,
    href: isHome ? link.href : `/${link.href}`,
  }));

  return (
    <header className="sticky top-0 z-50 border-b border-white/10 bg-[#0b0202]/90 backdrop-blur-xl">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-5 py-4 sm:px-8">
        <Link
          to="/"
          className="flex items-center gap-2.5"
          aria-label="SpecUse หน้าแรก"
          onClick={() => setMenuOpen(false)}
        >
          <span className="grid h-10 w-10 place-items-center rounded-xl bg-obsidian-900 text-white shadow-sm">
            <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" aria-hidden="true">
              <path d="M5 7.5h14M5 12h14M5 16.5h8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
              <circle cx="18" cy="16.5" r="2" fill="#f87171" />
            </svg>
          </span>
          <span className="text-xl font-extrabold tracking-tight text-slate-100">
            Spec<span className="text-ruby-600">Use</span>
          </span>
        </Link>

        <button
          type="button"
          className="grid h-11 w-11 place-items-center rounded-xl border border-white/15 text-slate-100 transition hover:bg-white/5 lg:hidden"
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

        <nav className="hidden items-center gap-6 lg:flex" aria-label="เมนูหลัก">
          <Link
            to="/build"
            className="text-sm font-bold text-ruby-200 transition hover:text-ruby-100"
          >
            ทดลองจัดสเปก
          </Link>
          {links.map((link) => (
            <a key={link.href} className="text-sm font-semibold text-slate-300 transition hover:text-ruby-300" href={link.href}>
              {link.label}
            </a>
          ))}
          <a
            href={isHome ? "#member" : "/#member"}
            className="rounded-full border border-white/15 px-5 py-2.5 text-sm font-bold text-slate-100 transition hover:border-ruby-400 hover:text-ruby-300"
          >
            ดูสิทธิ์สมาชิก
          </a>
          <a
            href={isHome ? "#public-builds" : "/#public-builds"}
            className="rounded-full bg-ruby-500 px-5 py-2.5 text-sm font-extrabold text-white transition hover:bg-ruby-400"
          >
            ดูตัวอย่างสเปก
          </a>
        </nav>
      </div>

      {menuOpen && (
        <nav className="border-t border-white/10 bg-[#0b0202] px-5 py-4 lg:hidden" aria-label="เมนูมือถือ">
          <div className="mx-auto flex max-w-7xl flex-col gap-1">
            <Link
              to="/build"
              className="rounded-lg px-3 py-3 font-bold text-ruby-200 transition hover:bg-white/5"
              onClick={() => setMenuOpen(false)}
            >
              ทดลองจัดสเปก
            </Link>
            {links.map((link) => (
              <a
                key={link.href}
                className="rounded-lg px-3 py-3 font-semibold text-slate-200 transition hover:bg-white/5"
                href={link.href}
                onClick={() => setMenuOpen(false)}
              >
                {link.label}
              </a>
            ))}
            <a
              href={isHome ? "#member" : "/#member"}
              className="mt-1 rounded-lg bg-obsidian-900 px-3 py-3 font-bold text-white"
              onClick={() => setMenuOpen(false)}
            >
              ดูสิทธิ์สมาชิก
            </a>
            <a
              href={isHome ? "#public-builds" : "/#public-builds"}
              className="mt-2 rounded-lg bg-ruby-500 px-3 py-3 font-extrabold text-white"
              onClick={() => setMenuOpen(false)}
            >
              ดูตัวอย่างสเปก
            </a>
          </div>
        </nav>
      )}
    </header>
  );
}
