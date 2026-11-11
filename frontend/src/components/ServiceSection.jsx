const accessLevels = [
  {
    label: "Guest",
    description: "เริ่มต้นจัดสเปกได้เลย ไม่ต้องสมัครสมาชิก",
    features: ["ดู ค้นหา และกรองอุปกรณ์", "เลือกชิ้นส่วนและดูราคารวม", "แชร์สเปกด้วยลิงก์"],
    featured: false,
  },
  {
    label: "Member",
    description: "มีบัญชีแล้ว จัดเก็บสเปกของคุณได้ครบ",
    features: ["บันทึกสเปกได้หลายชุด", "ตั้งชื่อและเขียนคำอธิบาย", "เลือกเปิดสาธารณะหรือส่วนตัว"],
    featured: true,
  },
];

export default function ServiceSection() {
  return (
    <section className="bg-white py-20 sm:py-24">
      <div className="mx-auto grid max-w-7xl gap-12 px-5 sm:px-8 lg:grid-cols-[0.9fr_1.1fr] lg:items-center lg:gap-20">
        <div>
          <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-aqua-600">เริ่มได้ตามสะดวก</p>
          <h2 className="mt-4 text-3xl font-black leading-tight tracking-tight text-navy-950 sm:text-4xl">
            จะลองจัดก่อน
            <br />
            หรือบันทึกไว้ก็ได้
          </h2>
          <p className="mt-5 max-w-lg leading-8 text-slate-600">
            SpecUse ออกแบบให้เริ่มต้นง่ายสำหรับทุกคน และมีเครื่องมือเพิ่มเติมสำหรับคนที่อยากจัดเก็บสเปกไว้กลับมาดูภายหลัง
          </p>
          <a href="#start" className="mt-7 inline-flex items-center gap-2 font-extrabold text-aqua-600 transition hover:gap-3">
            ดูวิธีเริ่มต้น
            <span aria-hidden="true">→</span>
          </a>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          {accessLevels.map((level) => (
            <article
              key={level.label}
              id={level.label === "Member" ? "member" : undefined}
              className={`rounded-3xl p-6 sm:p-7 ${level.featured ? "bg-navy-950 text-white shadow-card" : "border border-slate-100 bg-cream text-navy-950"}`}
            >
              <div className="flex items-center justify-between">
                <span className={`text-xs font-extrabold uppercase tracking-[0.2em] ${level.featured ? "text-aqua-400" : "text-aqua-600"}`}>สำหรับ</span>
                {level.featured && <span className="rounded-full bg-white/10 px-3 py-1 text-xs font-bold text-aqua-300">บันทึกสเปกได้</span>}
              </div>
              <h3 className="mt-2 text-3xl font-black">{level.label}</h3>
              <p className={`mt-3 min-h-14 text-sm leading-6 ${level.featured ? "text-white/65" : "text-slate-600"}`}>{level.description}</p>
              <ul className="mt-6 space-y-4">
                {level.features.map((feature) => (
                  <li key={feature} className="flex items-start gap-3 text-sm font-medium">
                    <span className={`mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full ${level.featured ? "bg-aqua-500/15 text-aqua-300" : "bg-aqua-500/10 text-aqua-600"}`}>
                      <svg viewBox="0 0 16 16" fill="none" className="h-3 w-3" aria-hidden="true">
                        <path d="m3.5 8 3 3 6-6" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                    </span>
                    <span className={level.featured ? "text-white/85" : "text-slate-700"}>{feature}</span>
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
