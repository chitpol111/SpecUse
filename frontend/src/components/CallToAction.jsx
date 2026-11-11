export default function CallToAction() {
  return (
    <section className="bg-[#0b0202] px-5 pb-20 sm:px-8 sm:pb-24">
      <div className="closing-panel glass-panel relative mx-auto max-w-7xl overflow-hidden rounded-[2rem] px-6 py-12 text-center sm:px-12 sm:py-16">
        <div className="relative mx-auto max-w-2xl">
          <h2 className="mt-4 text-3xl font-black leading-tight tracking-tight text-white sm:text-4xl">
            สเปกที่เหมาะกับคุณ
            <br className="hidden sm:block" /> เริ่มได้จากตรงนี้
          </h2>
          <p className="mx-auto mt-4 max-w-xl leading-7 text-white/85">
            ชมตัวอย่างแนวทางจัดสเปก ข้อมูลเพื่อประกอบการวางแผนเท่านั้น
          </p>
          <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
            <a href="#public-builds" className="rounded-full bg-ruby-500 px-7 py-3.5 font-extrabold text-white shadow-lg shadow-ruby-500/20 transition hover:-translate-y-0.5 hover:bg-ruby-400">
              ดูตัวอย่างแนวทาง
            </a>
            <a href="#member" className="rounded-full border border-white/20 px-7 py-3.5 font-extrabold text-white transition hover:border-ruby-300/50 hover:bg-white/5">
              ดูสิทธิ์สมาชิก
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}
