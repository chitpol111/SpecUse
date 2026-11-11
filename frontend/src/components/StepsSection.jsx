const steps = [
  { number: "01", title: "เลือกอุปกรณ์", text: "เลือกชิ้นส่วนที่เหมาะกับการใช้งานและงบของคุณ" },
  { number: "02", title: "ตรวจความเข้ากันได้", text: "เช็กความสัมพันธ์พื้นฐานของ CPU เมนบอร์ด และ RAM" },
  { number: "03", title: "ดูราคารวม", text: "เห็นยอดรวมของรายการที่เลือกก่อนตัดสินใจ" },
  { number: "04", title: "บันทึกหรือแชร์", text: "บันทึกเป็นสมาชิก หรือแชร์สเปกให้คนอื่นช่วยดู" },
];

export default function StepsSection() {
  return (
    <section id="start" className="overflow-hidden bg-navy-950 py-20 text-white sm:py-24">
      <div className="mx-auto max-w-7xl px-5 sm:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-aqua-400">ไม่กี่ขั้นตอนก็เห็นภาพ</p>
          <h2 className="mt-4 text-3xl font-black tracking-tight sm:text-4xl">จากไอเดีย สู่สเปกที่วางแผนได้</h2>
          <p className="mt-4 leading-7 text-white/60">มีลำดับช่วยเช็กให้ก่อนที่คุณจะตัดสินใจเลือกอุปกรณ์</p>
        </div>
        <div className="relative mt-14 grid gap-8 sm:grid-cols-2 lg:grid-cols-4 lg:gap-5">
          <div className="absolute left-[12%] right-[12%] top-7 hidden border-t border-dashed border-white/20 lg:block" aria-hidden="true" />
          {steps.map((step) => (
            <article key={step.number} className="relative">
              <span className="grid h-14 w-14 place-items-center rounded-2xl border border-aqua-400/30 bg-navy-900 text-sm font-black text-aqua-300 shadow-[0_0_30px_rgba(22,184,166,0.12)]">
                {step.number}
              </span>
              <h3 className="mt-6 text-lg font-extrabold">{step.title}</h3>
              <p className="mt-2 max-w-xs text-sm leading-7 text-white/60">{step.text}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
