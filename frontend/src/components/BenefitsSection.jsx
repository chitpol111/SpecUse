const benefits = [
  {
    number: "01",
    title: "เห็นราคารวมชัดเจน",
    text: "ติดตามยอดรวมของอุปกรณ์ที่เลือก และวางแผนให้อยู่ในงบได้ง่ายขึ้น",
    icon: (
      <>
        <path d="M5 7.5h14M5 12h14M5 16.5h8" />
        <circle cx="17.5" cy="16.5" r="1.5" />
      </>
    ),
  },
  {
    number: "02",
    title: "เช็กความเข้ากันได้",
    text: "ตรวจสอบเบื้องต้นระหว่าง Socket ของ CPU กับเมนบอร์ด และชนิด RAM",
    icon: (
      <>
        <path d="m5 12 4 4L19 6" />
        <path d="M20 12a8 8 0 1 1-4-6.93" />
      </>
    ),
  },
  {
    number: "03",
    title: "เก็บไว้หรือส่งต่อได้",
    text: "สมาชิกบันทึกสเปกได้หลายชุด ส่วนสเปกสาธารณะก็แชร์ให้คนอื่นดูได้",
    icon: (
      <>
        <circle cx="18" cy="5" r="2.5" />
        <circle cx="6" cy="12" r="2.5" />
        <circle cx="18" cy="19" r="2.5" />
        <path d="m8.2 10.8 7.6-4.6M8.2 13.2l7.6 4.6" />
      </>
    ),
  },
];

export default function BenefitsSection() {
  return (
    <section id="benefits" className="bg-cream py-20 sm:py-24">
      <div className="mx-auto max-w-7xl px-5 sm:px-8">
        <div className="flex flex-col justify-between gap-5 md:flex-row md:items-end">
          <div className="max-w-2xl">
            <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-aqua-600">ผู้ช่วยจัดสเปกของคุณ</p>
            <h2 className="mt-4 text-3xl font-black tracking-tight text-navy-950 sm:text-4xl">
              เลือกได้อย่างมั่นใจขึ้น
            </h2>
          </div>
          <p className="max-w-md leading-7 text-slate-600">
            รวมสิ่งที่ต้องเช็กไว้ในที่เดียว เพื่อช่วยให้การวางแผนประกอบคอมของคุณเป็นขั้นเป็นตอน
          </p>
        </div>
        <div className="mt-12 grid gap-5 lg:grid-cols-3">
          {benefits.map((benefit) => (
            <article key={benefit.number} className="rounded-3xl border border-slate-100 bg-white p-7 shadow-sm transition hover:-translate-y-1 hover:shadow-card sm:p-8">
              <div className="flex items-center justify-between">
                <span className="grid h-14 w-14 place-items-center rounded-2xl bg-aqua-500/10 text-aqua-600">
                  <svg viewBox="0 0 24 24" fill="none" className="h-7 w-7" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    {benefit.icon}
                  </svg>
                </span>
                <span className="text-sm font-bold tracking-widest text-slate-300">{benefit.number}</span>
              </div>
              <h3 className="mt-7 text-xl font-extrabold text-navy-900">{benefit.title}</h3>
              <p className="mt-3 leading-7 text-slate-600">{benefit.text}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
