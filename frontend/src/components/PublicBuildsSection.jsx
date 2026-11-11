const builds = [
  {
    useCase: "ทำงานทั่วไป",
    title: "เริ่มต้นทำงานและเรียน",
    description: "ตัวอย่างแนวทางจัดสเปกสำหรับงานเอกสาร เรียนออนไลน์ และใช้งานประจำวัน",
    parts: ["CPU", "เมนบอร์ด", "RAM", "Storage"],
    tone: "from-red-950 to-obsidian-900",
  },
  {
    useCase: "เล่นเกม",
    title: "เล่นเกมที่บ้าน",
    description: "ตัวอย่างแนวทางเลือกอุปกรณ์โดยคำนึงถึงประสบการณ์เล่นเกมและงบประมาณ",
    parts: ["CPU", "การ์ดจอ", "RAM", "PSU"],
    tone: "from-rose-950 to-obsidian-900",
  },
  {
    useCase: "สร้างสรรค์งาน",
    title: "ทำงานภาพและวิดีโอ",
    description: "ตัวอย่างแนวทางคำนึงถึงพลังประมวลผลและพื้นที่จัดเก็บสำหรับไฟล์งาน",
    parts: ["CPU", "RAM", "การ์ดจอ", "Storage"],
    tone: "from-red-900 to-obsidian-950",
  },
];

export default function PublicBuildsSection() {
  return (
    <section id="public-builds" className="bg-[#0b0202] py-20 sm:py-24">
      <div className="mx-auto max-w-7xl px-5 sm:px-8">
        <div className="flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-black tracking-tight text-slate-50 sm:text-4xl">ดูตัวอย่างแนวทางจัดสเปก</h2>
            <p className="mt-4 leading-7 text-slate-300">ตัวอย่างประกอบการวางแผน ไม่ใช่สเปกสมาชิกหรือรีวิวจากผู้ใช้จริง</p>
          </div>
          <span className="inline-flex w-fit items-center gap-2 rounded-full border border-ruby-400/20 bg-ruby-400/10 px-4 py-2 text-xs font-bold text-ruby-200">
            <span className="h-2 w-2 rounded-full bg-ruby-500" />
            ข้อมูลตัวอย่าง
          </span>
        </div>
        <div className="mt-10 grid gap-5 lg:grid-cols-3">
          {builds.map((build) => (
            <article key={build.title} className="glass-panel overflow-hidden rounded-3xl transition hover:-translate-y-1 hover:border-ruby-300/30 hover:shadow-card">
              <div className={`relative h-40 bg-gradient-to-br ${build.tone} p-6`}>
                <span className="inline-flex rounded-full border border-white/15 bg-obsidian-950/60 px-3 py-1.5 text-xs font-bold text-slate-100 backdrop-blur-md">{build.useCase}</span>
                <span className="absolute right-6 top-6 text-xs font-semibold text-slate-300">แนวทางตัวอย่าง</span>
                <svg viewBox="0 0 160 110" className="absolute bottom-4 right-5 h-24 w-36 text-rose-100/45 sm:right-6" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true">
                  <rect x="22" y="10" width="100" height="88" rx="8" />
                  <path d="M92 10v88M32 20h50v40H32zM32 68h50v20H32z" />
                  <rect x="43" y="30" width="18" height="18" rx="3" />
                  <path d="M68 29v20m7-20v20M39 74h36v8H39zM98 22h15m-15 6h15" />
                  <circle cx="107" cy="68" r="13" />
                  <circle cx="107" cy="68" r="5" />
                  <circle cx="107" cy="86" r="3" />
                  <path d="M103 68h8m-4-4v8" />
                </svg>
              </div>
              <div className="p-6 sm:p-7">
                <h3 className="text-lg font-extrabold text-slate-100">{build.title}</h3>
                <p className="mt-2 min-h-[3.5rem] text-sm leading-6 text-slate-300">{build.description}</p>
                <div className="mt-5 flex flex-wrap gap-2">
                  {build.parts.map((part) => (
                    <span key={part} className="glass-panel-soft rounded-lg px-2.5 py-1.5 text-xs font-semibold text-slate-200">{part}</span>
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
