export default function CallToAction() {
  return (
    <section className="bg-cream px-5 pb-20 sm:px-8 sm:pb-24">
      <div className="relative mx-auto max-w-7xl overflow-hidden rounded-[2rem] bg-aqua-500 px-6 py-12 text-center sm:px-12 sm:py-16">
        <div className="absolute -left-16 -top-32 h-72 w-72 rounded-full border-[40px] border-white/10" aria-hidden="true" />
        <div className="absolute -bottom-40 -right-16 h-80 w-80 rounded-full border-[50px] border-white/10" aria-hidden="true" />
        <div className="relative mx-auto max-w-2xl">
          <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-navy-950/65">เริ่มต้นได้เลย</p>
          <h2 className="mt-4 text-3xl font-black leading-tight tracking-tight text-navy-950 sm:text-4xl">
            สเปกที่เหมาะกับคุณ
            <br className="hidden sm:block" /> เริ่มได้จากตรงนี้
          </h2>
          <p className="mx-auto mt-4 max-w-xl leading-7 text-navy-950/75">
            ลองวางแผน เลือกอุปกรณ์ และดูภาพรวมของสเปกที่คุณอยากได้
          </p>
          <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
            <a href="#start" className="rounded-full bg-navy-950 px-7 py-3.5 font-extrabold text-white shadow-lg transition hover:-translate-y-0.5 hover:bg-navy-900">
              เริ่มจัดสเปก
            </a>
            <a href="#member" className="rounded-full border border-navy-950/20 px-7 py-3.5 font-extrabold text-navy-950 transition hover:bg-white/30">
              ดูสิทธิ์สมาชิก
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}
