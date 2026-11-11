import { useState } from "react";

const questions = [
  {
    question: "SpecUse ช่วยทำอะไรได้บ้าง?",
    answer: "ขณะนี้เว็บแสดงตัวอย่างแนวทางเท่านั้น เครื่องมือจัดรายการ ดูราคารวม และตรวจสอบความเข้ากันได้ยังอยู่ระหว่างพัฒนา",
  },
  {
    question: "ต้องสมัครสมาชิกก่อนเริ่มจัดสเปกไหม?",
    answer: "ระบบจัดสเปกและสมาชิกยังไม่เปิดให้ใช้งาน สิทธิ์ของ Guest และ Member ที่แสดงบนหน้านี้เป็นแนวทางที่วางแผนไว้",
  },
  {
    question: "ผลตรวจความเข้ากันได้ครอบคลุมทุกอย่างหรือไม่?",
    answer: "ระบบตรวจสอบยังอยู่ระหว่างพัฒนา โดยมีแผนให้ครอบคลุมข้อมูลเบื้องต้น เช่น Socket ของ CPU กับเมนบอร์ด และชนิด RAM ทั้งนี้ควรตรวจรายละเอียดอุปกรณ์อื่นก่อนซื้อด้วย",
  },
  {
    question: "ราคาที่แสดงเป็นราคาสุดท้ายจากร้านค้าหรือไม่?",
    answer: "ขณะนี้ยังไม่มีแคตตาล็อกสินค้าหรือราคาจากร้านค้า เมื่อมีข้อมูลดังกล่าวในอนาคต ราคาและความพร้อมจำหน่ายอาจเปลี่ยนแปลง ควรตรวจสอบกับผู้ขายก่อนสั่งซื้อ",
  },
];

export default function FAQSection() {
  const [openIndex, setOpenIndex] = useState(0);

  return (
    <section id="faq" className="bg-[#120505] py-20 sm:py-24">
      <div className="mx-auto grid max-w-7xl gap-12 px-5 sm:px-8 lg:grid-cols-[0.75fr_1.25fr] lg:gap-20">
        <div>
          <h2 className="text-3xl font-black tracking-tight text-slate-50 sm:text-4xl">มีข้อสงสัยใช่ไหม?</h2>
          <p className="mt-4 leading-7 text-slate-300">รวมคำตอบเบื้องต้นเกี่ยวกับการจัดสเปกและการใช้งาน SpecUse</p>
        </div>
        <div className="divide-y divide-white/10">
          {questions.map((item, index) => {
            const isOpen = index === openIndex;
            const answerId = `faq-answer-${index}`;
            return (
              <article key={item.question} className="py-2">
                <h3>
                  <button
                    type="button"
                    className="flex min-h-14 w-full items-center justify-between gap-5 py-5 text-left text-base font-extrabold text-slate-100 sm:text-lg"
                    aria-expanded={isOpen}
                    aria-controls={answerId}
                    onClick={() => setOpenIndex(isOpen ? -1 : index)}
                  >
                    {item.question}
                    <span className={`grid h-8 w-8 shrink-0 place-items-center rounded-full transition ${isOpen ? "rotate-45 bg-ruby-500 text-white" : "bg-obsidian-800 text-slate-300"}`} aria-hidden="true">
                      <svg viewBox="0 0 16 16" fill="none" className="h-4 w-4">
                        <path d="M8 3v10M3 8h10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
                      </svg>
                    </span>
                  </button>
                </h3>
                {isOpen && (
                  <div id={answerId} className="max-w-2xl pb-5 pr-12 text-sm leading-7 text-slate-300">
                    {item.answer}
                  </div>
                )}
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}
