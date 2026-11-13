const accessLevels = [
  {
    label: "Guest",
    description: "จัดสเปกเบื้องต้นได้โดยไม่ต้องเข้าสู่ระบบ",
    features: ["ดู ค้นหา และกรองอุปกรณ์", "เลือกชิ้นส่วนและดูราคารวม", "ตรวจความเข้ากันได้เบื้องต้น"],
    featured: false,
  },
  {
    label: "Member",
    description: "เข้าสู่ระบบเพื่อเก็บสเปกไว้ดูภายหลัง",
    features: ["บันทึกสเปกได้หลายชุด", "ตั้งชื่อและเขียนคำอธิบาย", "จัดการสเปกส่วนตัว"],
    featured: true,
  },
];

export default function ServiceSection() {
  return (
    <section className="bg-[#120505] py-20 sm:py-24">
      <div className="mx-auto grid max-w-7xl gap-12 px-5 sm:px-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center lg:gap-20">
        <div>
          <h2 className="text-3xl font-black leading-tight tracking-tight text-slate-50 sm:text-4xl">
            สิทธิ์การใช้งาน
            <br />
            สำหรับ Guest และ Member
          </h2>
          <p className="mt-5 max-w-lg leading-8 text-slate-300">
            ทดลองจัดสเปกได้ทันที หรือเข้าสู่ระบบเพื่อบันทึกสเปกส่วนตัว
          </p>
          <a href="#start" className="mt-7 inline-flex items-center gap-2 font-extrabold text-ruby-300 transition hover:gap-3">
            ดูวิธีเริ่มต้น
            <svg viewBox="0 0 20 20" fill="none" className="h-5 w-5" aria-hidden="true">
              <path d="M4 10h12m-5-5 5 5-5 5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </a>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {accessLevels.map((level) => (
            <article
              key={level.label}
              id={level.label === "Member" ? "member" : undefined}
              className={`glass-panel rounded-3xl p-6 sm:p-7 ${level.featured ? "ring-1 ring-ruby-400/20" : ""}`}
            >
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-2xl font-black">{`สำหรับ ${level.label}`}</h3>
                {level.featured && <span className="rounded-full bg-ruby-400/10 px-3 py-1 text-xs font-bold text-ruby-200">สำหรับสมาชิก</span>}
              </div>
              <p className="mt-3 min-h-14 text-sm leading-6 text-slate-300">{level.description}</p>
              <ul className="mt-6 space-y-4">
                {level.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3 text-sm font-medium">
                    <span className="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full bg-ruby-400/10 text-ruby-300">
                      <svg viewBox="0 0 16 16" fill="none" className="h-3 w-3" aria-hidden="true">
                        <path d="m3.5 8 3 3 6-6" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    </span>
                    <span className="text-slate-200">{feature}</span>
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
