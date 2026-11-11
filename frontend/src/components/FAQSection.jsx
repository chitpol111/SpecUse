import { useState } from "react";

const questions = [
  {
    question: "SpecUse ช่วยทำอะไรได้บ้าง?",
    answer: "ช่วยเลือกและจัดรายการอุปกรณ์คอมพิวเตอร์ตามการใช้งานและงบประมาณ ดูราคารวม และตรวจสอบความเข้ากันได้เบื้องต้นของ CPU, เมนบอร์ด และ RAM",
  },
  {
    question: "ต้องสมัครสมาชิกก่อนเริ่มจัดสเปกไหม?",
    answer: "ไม่จำเป็น Guest เริ่มเลือกอุปกรณ์และแชร์สเปกได้ ส่วนสมาชิกสามารถบันทึกสเปกหลายชุดและเลือกตั้งค่าเป็นสาธารณะหรือส่วนตัว",
  },
  {
    question: "ผลตรวจความเข้ากันได้ครอบคลุมทุกอย่างหรือไม่?",
    answer: "เป็นการตรวจสอบเบื้องต้นตามข้อมูลอุปกรณ์ เช่น Socket ของ CPU กับเมนบอร์ด และชนิด RAM กับเมนบอร์ด จึงควรตรวจรายละเอียดอุปกรณ์อื่นก่อนซื้อด้วย",
  },
  {
    question: "ราคาที่แสดงเป็นราคาสุดท้ายจากร้านค้าหรือไม่?",
    answer: "ระบบแสดงราคารวมจากข้อมูลสินค้าในระบบ ราคาและความพร้อมจำหน่ายจริงอาจเปลี่ยนแปลง ควรตรวจสอบกับผู้ขายก่อนสั่งซื้อ",
  },
];

export default function FAQSection() {
  const [openIndex, setOpenIndex] = useState(0);

  return (
    <section id="faq" className="bg-white py-20 sm:py-24">
      <div className="mx-auto grid max-w-7xl gap-12 px-5 sm:px-8 lg:grid-cols-[0.75fr_1.25fr] lg:gap-20">
        <div>
          <p className="text-sm font-extrabold uppercase tracking-[0.18em] text-aqua-600">คำถามที่พบบ่อย</p>
          <h2 className="mt-4 text-3xl font-black tracking-tight text-navy-950 sm:text-4xl">มีข้อสงสัยใช่ไหม?</h2>
          <p className="mt-4 leading-7 text-slate-600">รวมคำตอบเบื้องต้นเกี่ยวกับการจัดสเปกและการใช้งาน SpecUse</p>
        </div>
        <div className="divide-y divide-slate-100">
          {questions.map((item, index) => {
            const isOpen = index === openIndex;
            const answerId = `faq-answer-${index}`;
            return (
              <article key={item.question} className="py-2">
                <h3>
                  <button
                    type="button"
                    className="flex w-full items-center justify-between gap-5 py-5 text-left text-base font-extrabold text-navy-900 sm:text-lg"
                    aria-expanded={isOpen}
                    aria-controls={answerId}
                    onClick={() => setOpenIndex(isOpen ? -1 : index)}
                  >
                    {item.question}
                    <span className={`grid h-8 w-8 shrink-0 place-items-center rounded-full transition ${isOpen ? "rotate-45 bg-aqua-500 text-navy-950" : "bg-cream text-slate-500"}`} aria-hidden="true">
                      <svg viewBox="0 0 16 16" fill="none" className="h-4 w-4">
                        <path d="M8 3v10M3 8h10" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
                      </svg>
                    </span>
                  </button>
                </h3>
                {isOpen && (
                  <div id={answerId} className="max-w-2xl pb-5 pr-12 text-sm leading-7 text-slate-600">
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
