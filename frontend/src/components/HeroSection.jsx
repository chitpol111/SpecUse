export default function HeroSection() {
  return (
    <section id="home" className="relative overflow-hidden">
      <div className="hero-grid absolute inset-0" aria-hidden="true" />
      <div className="hero-orb absolute -right-32 top-0 h-[38rem] w-[38rem]" aria-hidden="true" />
      <div className="relative mx-auto grid max-w-7xl items-center gap-12 px-5 pb-16 pt-12 sm:px-8 sm:pb-24 sm:pt-16 lg:grid-cols-[1.02fr_0.98fr] lg:gap-14 lg:pb-28 lg:pt-20">
        <div className="fade-up max-w-2xl">
          <h1 className="text-[2.6rem] font-black leading-[1.12] tracking-[-0.035em] text-slate-50 sm:text-6xl lg:text-[4.25rem]">
            คอมที่ใช่
            <br />
            <span className="headline-accent mt-1 inline-block">
              เริ่มจากสเปกที่ใช่
            </span>
          </h1>
          <p className="mt-6 max-w-[34rem] text-base leading-8 text-slate-300 sm:text-lg">
            เลือกชิ้นส่วนตามการใช้งานและงบประมาณ พร้อมตรวจความเข้ากันได้เบื้องต้นและดูราคารวม
          </p>
          <p className="mt-3 text-sm leading-6 text-ruby-200">
            ราคาเป็นข้อมูลอ้างอิง และการตรวจยังไม่ครอบคลุมอุปกรณ์ทุกส่วน
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <a
              href="/build"
              className="inline-flex min-h-14 items-center justify-center gap-2 rounded-full bg-ruby-500 px-7 py-4 font-extrabold text-white shadow-lg shadow-ruby-500/20 transition hover:-translate-y-0.5 hover:bg-ruby-400"
            >
              เริ่มจัดสเปก
              <svg viewBox="0 0 20 20" fill="none" className="h-5 w-5" aria-hidden="true">
                <path d="M4 10h12m-5-5 5 5-5 5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </a>
            <a
              href="#start"
              className="inline-flex min-h-14 items-center justify-center rounded-full border border-white/15 bg-obsidian-900/80 px-7 py-4 font-bold text-slate-100 transition hover:border-ruby-400/50 hover:bg-obsidian-800"
            >
              ดูขั้นตอนการใช้งาน
            </a>
          </div>
          <div className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-3 text-sm font-medium text-slate-300">
            <span className="inline-flex items-center gap-2">
              <CheckMark /> คำนึงถึงงบประมาณ
            </span>
            <span className="inline-flex items-center gap-2">
              <CheckMark /> ตรวจสอบความเข้ากันได้เบื้องต้น
            </span>
          </div>
        </div>

        <div className="fade-up-delay relative mx-auto w-full max-w-[570px]">
          <div className="absolute -inset-5 rounded-[2.5rem] bg-ruby-500/[0.06] blur-2xl" aria-hidden="true" />
          <div className="glass-panel relative rounded-[2rem] p-4 sm:p-6">
            <div className="flex items-center justify-between border-b border-white/10 pb-4">
              <div>
                <h2 className="mt-1 text-lg font-extrabold text-slate-100">ทำงานและเล่นเกม</h2>
                <p className="mt-1 text-xs font-medium leading-5 text-ruby-200">ภาพจำลองเพื่อประกอบการวางแผน</p>
              </div>
              <span className="rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs font-bold text-slate-300">ภาพจำลอง</span>
            </div>
            <div className="space-y-2.5 py-4">
              <PartRow icon="CPU" name="หน่วยประมวลผล" detail="เลือกตามการใช้งาน" color="bg-red-400/10 text-red-300" />
              <PartRow icon="MB" name="เมนบอร์ด" detail="ตรวจสอบ Socket" color="bg-rose-400/10 text-rose-300" />
              <PartRow icon="RAM" name="หน่วยความจำ" detail="ตรวจสอบชนิด RAM" color="bg-[#f87171]/10 text-[#fca5a5]" />
              <PartRow icon="GPU" name="การ์ดจอ" detail="เลือกให้เหมาะกับงาน" color="bg-red-300/10 text-red-200" />
            </div>
            <div className="glass-panel-soft rounded-2xl p-5 text-white sm:p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-slate-300">เป้าหมายของสเปก</p>
                  <p className="mt-1 text-2xl font-black sm:text-3xl">สมดุลกับงบที่ตั้งไว้</p>
                </div>
                <div className="grid h-12 w-12 place-items-center rounded-2xl bg-ruby-500 text-white">
                  <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" aria-hidden="true">
                    <path d="M4 17 9 12l3 3 7-8m0 0h-5m5 0v5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </div>
              </div>
              <p className="mt-4 text-xs leading-5 text-slate-300">แนวคิด: ตรวจสอบเบื้องต้นและประเมินงบก่อนตัดสินใจ</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function CheckMark() {
  return (
    <svg viewBox="0 0 20 20" fill="none" className="h-4 w-4 text-ruby-400" aria-hidden="true">
      <path d="m4 10 4 4 8-8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function PartRow({ icon, name, detail, color }) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-obsidian-800/80 px-3.5 py-3">
      <span className={`grid h-10 w-10 shrink-0 place-items-center rounded-xl text-[10px] font-black ${color}`}>{icon}</span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-bold text-slate-100">{name}</span>
        <span className="mt-0.5 block text-xs text-slate-400">{detail}</span>
      </span>
      <span className="text-sm font-semibold text-slate-400">ตัวอย่าง</span>
    </div>
  );
}
