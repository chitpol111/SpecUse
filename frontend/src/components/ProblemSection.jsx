const problems = [
  {
    title: "ไม่แน่ใจว่าจะเลือกอะไร",
    description: "อุปกรณ์มีหลายรุ่น หลายราคา เลือกไม่ถูกว่าแบบไหนเหมาะกับงานของเรา",
    icon: "search",
  },
  {
    title: "กลัวซื้อมาแล้วใช้ด้วยกันไม่ได้",
    description: "สเปกแต่ละชิ้นมีรายละเอียดที่ต้องเช็ก ทั้ง Socket และชนิดหน่วยความจำ",
    icon: "compatibility",
  },
  {
    title: "งบบานปลายโดยไม่รู้ตัว",
    description: "รวมราคาทีละชิ้นได้ยาก และอาจเผลอเลือกเกินงบที่ตั้งไว้",
    icon: "budget",
  },
];

export default function ProblemSection() {
  return (
    <section className="bg-[#120505] py-20 sm:py-24">
      <div className="mx-auto grid max-w-7xl gap-10 px-5 sm:px-8 lg:grid-cols-[0.8fr_1.2fr] lg:gap-20">
        <div className="max-w-2xl">
          <h2 className="text-3xl font-black tracking-tight text-slate-50 sm:text-4xl">
            ประกอบคอมไม่ควรเป็นเรื่องชวนปวดหัว
          </h2>
          <p className="mt-4 leading-7 text-slate-300">
            ก่อนจะได้คอมที่ตอบโจทย์ หลายคนต้องผ่านคำถามมากมายตั้งแต่ชิ้นแรกจนถึงชิ้นสุดท้าย
          </p>
        </div>
        <div className="divide-y divide-ruby-500/15">
          {problems.map((problem) => (
            <article key={problem.title} className="flex gap-5 py-6 first:pt-0 last:pb-0">
              <span className="grid h-12 w-12 shrink-0 place-items-center text-ruby-300">
                <ProblemIcon name={problem.icon} />
              </span>
              <div>
                <h3 className="text-xl font-extrabold text-slate-100">{problem.title}</h3>
                <p className="mt-2 leading-7 text-slate-300">{problem.description}</p>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function ProblemIcon({ name }) {
  const paths = {
    search: <><circle cx="10.8" cy="10.8" r="6.3" /><path d="m16 16 4 4" /></>,
    compatibility: <><rect x="4" y="4" width="6" height="6" rx="1" /><rect x="14" y="14" width="6" height="6" rx="1" /><path d="M10 7h3a2 2 0 0 1 2 2v5" /><path d="m12 12 3 3 3-3" /></>,
    budget: <><path d="M12 3v18" /><path d="M17 7.5c0-1.4-1.7-2.5-4-2.5S9 6.1 9 7.5 10.8 10 13 10s4 1.1 4 2.5-1.8 2.5-4 2.5-4-1.1-4-2.5" /></>,
  };

  return (
    <svg viewBox="0 0 24 24" fill="none" className="h-6 w-6" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {paths[name]}
    </svg>
  );
}
