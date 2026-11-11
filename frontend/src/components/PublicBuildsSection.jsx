const builds = [
  {
    useCase: "ทำงานทั่วไป",
    title: "เริ่มต้นทำงานและเรียน",
    description: "ตัวอย่างแนวทางจัดสเปกสำหรับงานเอกสาร เรียนออนไลน์ และใช้งานประจำวัน",
    parts: ["CPU", "เมนบอร์ด", "RAM", "Storage"],
    tone: "from-sky-50 to-white",
    icon: "01",
  },
  {
    useCase: "เล่นเกม",
    title: "เล่นเกมที่บ้าน",
    description: "ตัวอย่างแนวทางเลือกอุปกรณ์โดยคำนึงถึงประสบการณ์เล่นเกมและงบประมาณ",
    parts: ["CPU", "การ์ดจอ", "RAM", "PSU"],
    tone: "from-violet-50 to-white",
    icon: "02",
  },
  {
    useCase: "สร้างสรรค์งาน",
    title: "ทำงานภาพและวิดีโอ",
    description: "ตัวอย่างแนวทางคำนึงถึงพลังประมวลผลและพื้นที่จัดเก็บสำหรับไฟล์งาน",
    parts: ["CPU", "RAM", "การ์ดจอ", "Storage"],
    tone: "from-emerald-50 to-white",
    icon: "03",
  },
];

export default function PublicBuildsSection() {
  return (
    <section id="public-builds" className="bg-cream py-20 sm:py-24">
      <div className="mx-auto max-w-7xl px-5 sm:px-8">
        <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
          <div className="max-w-2xl">
            <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-aqua-600">ไอเดียเริ่มต้น</p>
            <h2 className="mt-4 text-3xl font-black tracking-tight text-navy-950 sm:text-4xl">ดูตัวอย่างแนวทางจัดสเปก</h2>
            <p className="mt-4 leading-7 text-slate-600">ตัวอย่างประกอบการวางแผน ไม่ใช่สเปกสมาชิกหรือรีวิวจากผู้ใช้จริง</p>
          </div>
          <span className="inline-flex w-fit items-center gap-2 rounded-full border border-aqua-500/20 bg-white px-4 py-2 text-xs font-bold text-aqua-700">
            <span className="h-2 w-2 rounded-full bg-aqua-500" />
            ข้อมูลตัวอย่าง
          </span>
        </div>
        <div className="mt-10 grid gap-5 lg:grid-cols-3">
          {builds.map((build) => (
            <article key={build.icon} className="overflow-hidden rounded-3xl border border-slate-100 bg-white shadow-sm transition hover:-translate-y-1 hover:shadow-card">
              <div className={`relative h-36 bg-gradient-to-br ${build.tone} p-6`}>
                <div className="absolute right-6 top-5 text-6xl font-black tracking-tighter text-navy-900/[0.06]">{build.icon}</div>
                <span className="inline-flex rounded-full border border-white bg-white/80 px-3 py-1.5 text-xs font-bold text-navy-800">{build.useCase}</span>
                <div className="absolute bottom-5 left-6 flex items-end gap-2" aria-hidden="true">
                  <span className="h-9 w-9 rounded-lg border border-navy-900/10 bg-white/90 shadow-sm" />
                  <span className="h-14 w-12 rounded-t-xl border border-navy-900/10 bg-white/90 shadow-sm" />
                  <span className="h-11 w-9 rounded-lg border border-navy-900/10 bg-white/90 shadow-sm" />
                  <span className="h-7 w-7 rounded-lg border border-navy-900/10 bg-aqua-500/20 shadow-sm" />
                </div>
              </div>
              <div className="p-6">
                <h3 className="text-lg font-extrabold text-navy-900">{build.title}</h3>
                <p className="mt-2 min-h-[3.5rem] text-sm leading-6 text-slate-600">{build.description}</p>
                <div className="mt-5 flex flex-wrap gap-2">
                  {build.parts.map((part) => (
                    <span key={part} className="rounded-lg bg-cream px-2.5 py-1.5 text-xs font-semibold text-slate-600">{part}</span>
                  ))}
                </div>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
