export default function HeroSection() {
  return (
    <section id="home" className="relative overflow-hidden">
      <div className="hero-grid absolute inset-0" aria-hidden="true" />
      <div className="hero-orb absolute -right-32 top-0 h-[34rem] w-[34rem]" aria-hidden="true" />
      <div className="relative mx-auto grid max-w-7xl items-center gap-14 px-5 pb-20 pt-16 sm:px-8 sm:pb-24 sm:pt-20 lg:grid-cols-[1.05fr_0.95fr] lg:gap-10 lg:pb-28 lg:pt-24">
        <div className="fade-up max-w-2xl">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-aqua-500/20 bg-white/80 px-3.5 py-2 text-sm font-semibold text-navy-800 shadow-sm">
            <span className="h-2 w-2 rounded-full bg-aqua-500" />
            จัดสเปกคอมให้ง่ายขึ้นในแบบของคุณ
          </div>
          <h1 className="text-4xl font-black leading-[1.2] tracking-tight text-navy-950 sm:text-5xl lg:text-6xl">
            คอมที่ใช่
            <br />
            <span className="relative inline-block text-aqua-600">
              เริ่มจากสเปกที่ใช่
              <span className="absolute -bottom-1 left-0 -z-10 h-3 w-full rounded-full bg-aqua-500/15" />
            </span>
          </h1>
          <p className="mt-6 max-w-xl text-base leading-8 text-slate-600 sm:text-lg">
            เลือกชิ้นส่วนให้เหมาะกับสิ่งที่คุณทำและงบที่มี เช็กความเข้ากันได้เบื้องต้น พร้อมเห็นราคารวมก่อนตัดสินใจ
          </p>
          <div className="mt-9 flex flex-col gap-3 sm:flex-row">
            <a
              href="#start"
              className="inline-flex items-center justify-center gap-2 rounded-full bg-aqua-500 px-7 py-4 font-extrabold text-navy-950 shadow-lg shadow-aqua-500/20 transition hover:-translate-y-0.5 hover:bg-aqua-600 hover:text-white"
            >
              เริ่มจัดสเปก
              <svg viewBox="0 0 20 20" fill="none" className="h-5 w-5" aria-hidden="true">
                <path d="M4 10h12m-5-5 5 5-5 5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </a>
            <a
              href="#public-builds"
              className="inline-flex items-center justify-center rounded-full border border-navy-900/15 bg-white/70 px-7 py-4 font-bold text-navy-900 transition hover:border-navy-900/30 hover:bg-white"
            >
              ดูตัวอย่างสเปก
            </a>
          </div>
          <div className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-3 text-sm font-medium text-slate-500">
            <span className="inline-flex items-center gap-2">
              <CheckMark /> เหมาะกับงบของคุณ
            </span>
            <span className="inline-flex items-center gap-2">
              <CheckMark /> เช็กความเข้ากันได้
            </span>
          </div>
        </div>

        <div className="fade-up-delay relative mx-auto w-full max-w-[550px]">
          <div className="absolute -left-7 top-16 h-28 w-28 rounded-full bg-aqua-500/10 blur-2xl" aria-hidden="true" />
          <div className="absolute -right-3 bottom-10 h-36 w-36 rounded-full bg-sky-300/20 blur-3xl" aria-hidden="true" />
          <div className="relative rounded-[2rem] border border-white bg-white/80 p-4 shadow-card backdrop-blur sm:p-6">
            <div className="flex items-center justify-between border-b border-slate-100 pb-4">
              <div>
                <p className="text-xs font-bold uppercase tracking-[0.16em] text-aqua-600">ตัวอย่างการจัดสเปก</p>
                <h2 className="mt-1 text-lg font-extrabold text-navy-900">ทำงานและเล่นเกม</h2>
              </div>
              <span className="rounded-full bg-emerald-50 px-3 py-1.5 text-xs font-bold text-emerald-700">ตัวอย่าง</span>
            </div>
            <div className="space-y-2.5 py-4">
              <PartRow icon="CPU" name="หน่วยประมวลผล" detail="เลือกตามการใช้งาน" color="bg-violet-50 text-violet-700" />
              <PartRow icon="MB" name="เมนบอร์ด" detail="ตรวจสอบ Socket" color="bg-sky-50 text-sky-700" />
              <PartRow icon="RAM" name="หน่วยความจำ" detail="ตรวจสอบชนิด RAM" color="bg-amber-50 text-amber-700" />
              <PartRow icon="GPU" name="การ์ดจอ" detail="เลือกให้เหมาะกับงาน" color="bg-rose-50 text-rose-700" />
            </div>
            <div className="rounded-2xl bg-navy-950 p-5 text-white sm:p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-white/65">งบประมาณตัวอย่าง</p>
                  <p className="mt-1 text-2xl font-black sm:text-3xl">เลือกได้ตามงบ</p>
                </div>
                <div className="grid h-12 w-12 place-items-center rounded-2xl bg-aqua-500 text-navy-950">
                  <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" aria-hidden="true">
                    <path d="M4 17 9 12l3 3 7-8m0 0h-5m5 0v5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </div>
              </div>
              <div className="mt-5 h-2 overflow-hidden rounded-full bg-white/15">
                <div className="h-full w-2/3 rounded-full bg-aqua-500" />
              </div>
              <p className="mt-2 text-xs text-white/55">ปรับรายการได้ พร้อมดูราคารวมก่อนตัดสินใจ</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

function CheckMark() {
  return (
    <svg viewBox="0 0 20 20" fill="none" className="h-4 w-4 text-aqua-600" aria-hidden="true">
      <path d="m4 10 4 4 8-8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function PartRow({ icon, name, detail, color }) {
  return (
    <div className="flex items-center gap-3 rounded-xl border border-slate-100 bg-white px-3.5 py-3">
      <span className={`grid h-10 w-10 shrink-0 place-items-center rounded-xl text-[10px] font-black ${color}`}>{icon}</span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-bold text-navy-900">{name}</span>
        <span className="mt-0.5 block text-xs text-slate-500">{detail}</span>
      </span>
      <span className="text-sm font-semibold text-slate-400">เลือก</span>
    </div>
  );
}
