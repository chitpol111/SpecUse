const benefits = [
  {
    title: "วางแผนราคารวม",
    text: "รวมราคาของอุปกรณ์ที่เลือก เพื่อเทียบกับงบประมาณได้ง่ายขึ้น",
    icon: (
      <>
        <path d="M5 7.5h14M5 12h14M5 16.5h8" />
        <circle cx="17.5" cy="16.5" r="1.5" />
      </>
    ),
  },
  {
    title: "ตรวจสอบความเข้ากันได้เบื้องต้น",
    text: "ตรวจ Socket ของ CPU กับเมนบอร์ด และชนิด RAM เบื้องต้น",
    icon: (
      <>
        <path d="m5 12 4 4L19 6" />
        <path d="M20 12a8 8 0 1 1-4-6.93" />
      </>
    ),
  },
  {
    title: "บันทึกสเปกส่วนตัว",
    text: "สมาชิกสามารถบันทึกหลายชุด ตั้งชื่อและเพิ่มคำอธิบายได้",
    icon: (
      <>
        <path d="M6 4h10l2 2v14H6zM9 4v5h6V4M9 20v-7h6v7" />
      </>
    ),
  },
];

export default function BenefitsSection() {
  return (
    <section id="benefits" className="bg-[#0b0202] py-20 sm:py-24">
      <div className="mx-auto max-w-7xl px-5 sm:px-8">
        <div className="flex flex-col justify-between gap-5 md:flex-row md:items-end">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-black tracking-tight text-slate-50 sm:text-4xl">
              เครื่องมือช่วยวางแผนสเปก
            </h2>
          </div>
          <p className="max-w-md leading-7 text-slate-300">
            เลือกอุปกรณ์ ตรวจสอบเบื้องต้น และบันทึกสเปกไว้ดูภายหลัง
          </p>
        </div>
        <div className="mt-12 grid gap-8 lg:grid-cols-[1.05fr_0.95fr] lg:gap-14">
          {benefits.slice(0, 1).map((benefit) => (
            <article key={benefit.title} className="glass-panel flex min-h-72 flex-col justify-between rounded-3xl p-7 sm:p-9">
              <span className="grid h-14 w-14 place-items-center rounded-2xl bg-ruby-400/10 text-ruby-300">
                <svg viewBox="0 0 24 24" fill="none" className="h-7 w-7" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  {benefit.icon}
                </svg>
              </span>
              <div className="mt-10">
                <h3 className="text-2xl font-extrabold text-slate-100 sm:text-3xl">{benefit.title}</h3>
                <p className="mt-3 max-w-lg leading-7 text-slate-300">{benefit.text}</p>
              </div>
            </article>
          ))}
          <div className="flex flex-col divide-y divide-ruby-500/15">
            {benefits.slice(1).map((benefit) => (
              <article key={benefit.title} className="flex gap-5 py-6 first:pt-0 last:pb-0">
                <span className="grid h-12 w-12 shrink-0 place-items-center rounded-2xl bg-ruby-400/10 text-ruby-300">
                  <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    {benefit.icon}
                  </svg>
                </span>
                <div>
                  <h3 className="text-lg font-extrabold text-slate-100">{benefit.title}</h3>
                  <p className="mt-2 leading-7 text-slate-300">{benefit.text}</p>
                </div>
              </article>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
