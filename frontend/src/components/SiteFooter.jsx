import { Link, useLocation } from "react-router-dom";

const footerLinks = [
  { label: "หน้าแรก", href: "#home" },
  { label: "จุดเด่น", href: "#benefits" },
  { label: "ตัวอย่างสเปก", href: "#public-builds" },
  { label: "คำถามที่พบบ่อย", href: "#faq" },
];

export default function SiteFooter() {
  const { pathname } = useLocation();
  const isHome = pathname === "/";

  return (
    <footer className="bg-obsidian-950 text-white">
      <div className="mx-auto max-w-7xl px-5 py-12 sm:px-8 sm:py-14">
        <div className="grid gap-10 md:grid-cols-[1.3fr_0.7fr]">
          <div className="max-w-md">
            <Link to="/" className="inline-flex items-center gap-2.5" aria-label="SpecUse กลับหน้าแรก">
              <span className="grid h-10 w-10 place-items-center rounded-xl bg-white/10 text-white">
                <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" aria-hidden="true">
                  <path d="M5 7.5h14M5 12h14M5 16.5h8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                  <circle cx="18" cy="16.5" r="2" fill="#f87171" />
                </svg>
              </span>
              <span className="text-xl font-extrabold tracking-tight">Spec<span className="text-ruby-400">Use</span></span>
            </Link>
            <p className="mt-4 leading-7 text-white/70">
              โปรเจกต์เพื่อการศึกษาที่กำลังพัฒนาเครื่องมือช่วยวางแผนสเปกคอมพิวเตอร์
            </p>
            <p className="mt-3 text-xs leading-6 text-white/60">
              ขณะนี้แสดงเฉพาะตัวอย่างแนวทาง ระบบจัดสเปกและสมาชิกยังไม่พร้อมใช้งาน
            </p>
          </div>
          <div>
            <h2 className="text-sm font-extrabold text-white/90">ลิงก์ที่เกี่ยวข้อง</h2>
            <nav className="mt-4 grid grid-cols-2 gap-x-6 gap-y-3" aria-label="ลิงก์ท้ายหน้า">
              {footerLinks.map((link) => (
                <a
                  key={link.href}
                  href={isHome ? link.href : `/${link.href}`}
                  className="text-sm text-white/70 transition hover:text-ruby-300"
                >
                  {link.label}
                </a>
              ))}
            </nav>
          </div>
        </div>
        <div className="mt-10 flex flex-col gap-3 border-t border-white/10 pt-6 text-xs text-white/60 sm:flex-row sm:items-center sm:justify-between">
          <p>© {new Date().getFullYear()} SU SpecUse</p>
          <p>เลือกให้เหมาะ เช็กให้ชัวร์ วางแผนให้อยู่ในงบ</p>
        </div>
      </div>
    </footer>
  );
}
