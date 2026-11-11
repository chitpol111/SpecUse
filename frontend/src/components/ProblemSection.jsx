const problems = [
  {
    number: "01",
    title: "ไม่แน่ใจว่าจะเลือกอะไร",
    description: "อุปกรณ์มีหลายรุ่น หลายราคา เลือกไม่ถูกว่าแบบไหนเหมาะกับงานของเรา",
    icon: "⌕",
  },
  {
    number: "02",
    title: "กลัวซื้อมาแล้วใช้ด้วยกันไม่ได้",
    description: "สเปกแต่ละชิ้นมีรายละเอียดที่ต้องเช็ก ทั้ง Socket และชนิดหน่วยความจำ",
    icon: "◇",
  },
  {
    number: "03",
    title: "งบบานปลายโดยไม่รู้ตัว",
    description: "รวมราคาทีละชิ้นได้ยาก และอาจเผลอเลือกเกินงบที่ตั้งไว้",
    icon: "฿",
  },
];

export default function ProblemSection() {
  return (
    <section className="bg-white py-20 sm:py-24">
      <div className="mx-auto max-w-7xl px-5 sm:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-aqua-600">คุ้นกับปัญหาเหล่านี้ไหม?</p>
          <h2 className="mt-4 text-3xl font-black tracking-tight text-navy-950 sm:text-4xl">
            ประกอบคอมไม่ควรเป็นเรื่องชวนปวดหัว
          </h2>
          <p className="mt-4 leading-7 text-slate-600">
            ก่อนจะได้คอมที่ตอบโจทย์ หลายคนต้องผ่านคำถามมากมายตั้งแต่ชิ้นแรกจนถึงชิ้นสุดท้าย
          </p>
        </div>
        <div className="mt-12 grid gap-5 md:grid-cols-3">
          {problems.map((problem) => (
            <article key={problem.number} className="group rounded-3xl border border-slate-100 bg-cream p-7 transition hover:-translate-y-1 hover:border-aqua-500/30 hover:shadow-card sm:p-8">
              <div className="flex items-start justify-between">
                <span className="grid h-12 w-12 place-items-center rounded-2xl bg-white text-2xl font-semibold text-aqua-600 shadow-sm">{problem.icon}</span>
                <span className="text-sm font-bold tracking-widest text-slate-300">{problem.number}</span>
              </div>
              <h3 className="mt-7 text-xl font-extrabold text-navy-900">{problem.title}</h3>
              <p className="mt-3 leading-7 text-slate-600">{problem.description}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
