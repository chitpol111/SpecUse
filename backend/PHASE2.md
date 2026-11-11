# Phase 2: schema, repository และ validation ภายใน

งานนี้มี domain/service และ unit tests สำหรับ compatibility และ RAM selection
เชื่อม `POST /api/v1/builds/validate` แล้ว ดู [HTTP contract และตัวอย่าง](BUILD_API.md)
GET Catalog ยังคง 6 fields เดิม และยังไม่มี build persistence หรือ Supabase adapter

## ข้อตกลงล่าสุดที่ยืนยันแล้ว

- CPU และ motherboard ใช้ `specs.cpu_socket`
- RAM และ motherboard ใช้ `specs.ram_type`
- RAM ใช้ `specs.modules_per_package` เป็น 1 หรือ 2 และ `ram_model_id` **ระดับเดียวกับ `id`**
- category เป็นตัวพิมพ์เล็ก แต่ยังไม่สร้าง enum สำหรับหมวดทั้งหมด
- ราคาเป็นจำนวนเต็มเงินบาทและยอดรวมคิดเฉพาะของที่ต้องซื้อเพิ่ม
- Build ที่ไม่ครบอนุญาตให้เป็น private draft ได้ ไม่สมมติหมวดที่จำเป็นเอง
- ยังไม่เปิดเผยแพร่ public จนกว่าจะตกลงเกณฑ์ความครบกับ Fluke

ตรวจแล้ว field ใน Go/JSON/SQL เดิมใช้ `is_active` ตรงกัน ไม่มี field `active`
ที่ต้องเปลี่ยนชื่อ คำว่า active ใน spec เป็นคำอธิบายสถานะ

## Schema ที่เตรียมไว้

ใช้ `001_products.up.sql` เดิม และเพิ่ม `002_compatibility.up.sql` แบบ additive:

| ตาราง | Fields/constraints หลัก |
|---|---|
| `products` | 6 fields เดิม; เพิ่ม CHECK category เป็น lowercase/ไม่มีช่องว่างหัวท้าย และ unique `(id, category)` สำหรับ foreign keys |
| `product_cpus` | product_id UUID PK/FK, category คงที่ `cpu`, cpu_socket ไม่ว่าง |
| `product_motherboards` | product_id UUID PK/FK, category คงที่ `motherboard`, cpu_socket/ram_type ไม่ว่าง |
| `ram_models` | id UUID PK, ram_type ไม่ว่าง, unique `(id, ram_type)` |
| `product_rams` | product_id UUID PK/FK, category คงที่ `ram`, ram_model_id, ram_type, modules_per_package IN (1, 2) |

Composite foreign keys ป้องกันการผูก CPU specs กับสินค้า category อื่น และป้องกัน
RAM SKU มี ram_type ต่างจาก RAM model ของตัวเอง ราคา/สถานะขายยังมาจาก `products`
เปิด RLS ให้ตารางใหม่โดยยังไม่เพิ่ม policies สำหรับ anon/member

ไม่ backfill socket/type หรือแปลง category ของข้อมูลเก่าแบบเงียบ ๆ หากมี category
ตัวพิมพ์ใหญ่ migration จะ fail เพื่อให้ตรวจข้อมูลก่อน สินค้าเดิมอาจยังไม่มี typed row;
Phase 2 repository จะไม่ยอมรับ CPU/motherboard/RAM ที่ metadata ไม่ครบ
foreign keys ยังไม่บังคับว่าทุก product ต้องมี typed row เพราะต้องทยอยเพิ่ม metadata เดิม

Migration ยังไม่ได้ execute กับ PostgreSQL/Supabase จริง ต้องตรวจบนฐาน local/test ก่อน
จาก `backend/` และฐานทดสอบว่างที่มี `gen_random_uuid()`:

```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 --single-transaction -f migrations/001_products.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 --single-transaction -f migrations/002_compatibility.up.sql
```

หากรัน 001 แล้ว ให้รันเฉพาะ 002 โปรแกรม API ไม่อ่าน `DATABASE_URL` และไม่รัน migrations เอง

### แบบร่าง schema สำหรับ private drafts ในระยะถัดไป

ตารางด้านล่างเป็น design เท่านั้น ยังไม่มี migration หรือ save API ในรอบนี้:

| ตาราง | ข้อมูลและ invariant ที่วางไว้ |
|---|---|
| `builds` | UUID id, owner อ้าง Supabase user, name, description, visibility ค่าเริ่มต้น private; ยังไม่เปิด public writes |
| `build_items` | build_id/product_id UUID FK, quantity > 0, 0 <= owned_quantity <= quantity; ไม่ใช้แทน RAM module requirements |
| `build_ram_items` | build_id/ram_model_id UUID FK, required_modules > 0, 0 <= owned_modules <= required_modules |

ไม่เก็บราคาหรือผล compatibility จาก client เป็นความจริง การบันทึก draft ต้องยังตรวจ
รูปแบบ input และสิทธิ์เจ้าของ แต่ไม่บังคับว่ามีทุกหมวดหรือเข้ากันได้ทั้งหมด
การเปลี่ยนไป public ต้องถูกปิดไว้ก่อน ไม่ใช้ `Compatible=true` เพียงค่าเดียวอนุมัติ
การลบบัญชีต้องทำให้ owner ไม่ระบุตัวตนและ build เป็น private ตาม spec;
รายละเอียด FK/transaction/trigger และ authorization จะทำพร้อม Auth/build persistence

## Repository และ domain

- `build.Repository.LoadSnapshot(ctx)` อ่าน products/specs/prices/status และ RAM models
  จากข้อมูลที่ Backend เชื่อถือได้ใน snapshot เดียว Database adapter ในอนาคตต้องใช้
  consistent read/transaction เพื่อไม่ให้ราคาและ specs มาจากคนละช่วงเวลา
- `repository.NewBuildMemory` รองรับทั้ง Catalog และ build repository จากข้อมูลชุดเดียว
  ตรวจ UUID, ราคา, lowercase category, typed metadata,
  module packaging, duplicate IDs และ RAM model/type relationship
- เก็บ inactive ไว้ใน snapshot เพื่อให้แยก unavailable ได้ แต่ RAM selector ใช้เฉพาะ active
- copy ทั้งตอนสร้างและคืน snapshot เพื่อกัน caller แก้ราคา/specs ของ repository
- executable ใช้ BuildMemory ซึ่ง delegate Catalog reads ผ่าน Memory เดิม;
  local fixture เพิ่ม typed metadata และ derive RAM models จาก model ID/type ของ SKU ในไฟล์
- `build.Service.Validate` รับเฉพาะ references/counts ไม่มี client price, specs, role หรือ compatibility result
- non-RAM คิด `(quantity - owned_quantity) * catalog price`; RAM คิดจากแพ็กที่ต้องซื้อ
  ตรวจ integer overflow ทั้งการคูณ/บวก ไม่ wrap เป็นยอดติดลบ
- RAM model ที่เป็นของอยู่แล้วก็ต้องเทียบ RAM type กับ motherboard แม้ไม่มีแพ็กต้องซื้อ

## Compatibility และ draft evaluation

ตรวจ CPU socket กับ motherboard socket และ RAM model type กับ motherboard type เท่านั้น
ถ้าไม่มี CPU/board/RAM ยังประเมินรายการที่มีได้ โดยไม่เพิ่ม required-category issues
ถ้ามี metadata ว่างต้องไม่รายงานว่าเข้ากันได้จากการเทียบค่าว่างกับค่าว่าง

`Compatible=true` หมายถึงไม่มีปัญหาจากสอง checks ที่ตกลงไว้ **ไม่ได้แปลว่า build ครบ**
Empty/partial draft จึงอาจได้ true และยอด 0 ซึ่งห้ามนำไปใช้เป็น publish gate
Go types ใน `internal/build` ยังเป็นภายใน Handler map เป็น HTTP DTO ที่ยืนยันกับ Fluke แล้ว

## RAM selection

คำนวณ `need = required_modules - owned_modules` และใช้ SKU ของ RAM model เดียวกันเท่านั้น:

1. ถ้า need เป็น 0 ไม่ซื้อแพ็ก ไม่ต้องมี active SKU และราคาเป็น 0
2. เลือกจำนวนโมดูลพอดีก่อน แม้แพ็กที่ทำให้เกินจะราคาถูกกว่า
3. ถ้าทำพอดีไม่ได้ เลือกจำนวนส่วนเกินน้อยที่สุด
4. ภายในจำนวนโมดูลเดียวกัน เลือกราคารวมต่ำที่สุด
5. ราคาเท่ากันเลือกจำนวนแพ็กน้อยที่สุด; SKU ขนาดเดียวกันราคาเท่ากันเลือก UUID
   พิมพ์เล็กที่เรียงมาก่อน และคืนรายการแพ็กเรียง UUID

ไม่มี stock limit ใน contract จึงใช้ SKU เดิมซ้ำได้ ไม่มีข้อมูล stock ถูกเพิ่มเอง
สำหรับแพ็ก 1/2 โมดูล การเก็บ SKU ที่ถูกที่สุดของแต่ละขนาดก็เพียงพอ:
ถ้ามีทั้งสองขนาด เปรียบเทียบราคาแพ็กคู่กับสองแพ็กเดี่ยวเพื่อเลือก exact combination
ถ้ามีเฉพาะแพ็กคู่ ใช้ ceil(need/2) แพ็ก อัลกอริทึมไม่จัดสรร array ตามจำนวนโมดูล

กฎ tie-break ข้อ 5 ระบุชัดใน tests และ HTTP contract ที่เปิดใช้ logic ชุดนี้

## Tests

```sh
go test ./...
go vet ./...
```

ใช้ memory/stub repository ไม่มี production Supabase หรือ secrets ครอบคลุม:

- socket/type ตรงและไม่ตรง, missing metadata, empty/partial draft
- owned quantities และราคาปัจจุบันจาก snapshot, RAM ที่ owned แล้วยังมีผลต่อ compatibility
- exact count ก่อนราคา, minimal excess, ราคาต่ำสุด, inactive/ต่างรุ่น, deterministic ties
- exhaustive oracle ตรวจทุก combination ขนาดเล็กของแพ็กเดี่ยว/คู่ รวมราคา 0 และกรณีมีเพียงขนาดเดียว
- จำนวนโมดูลขนาดใหญ่โดยไม่ใช้ memory ตามจำนวน, integer overflow, invalid counts และ references
- repository data integrity, copy isolation, canceled context และ service/repository integration
- tests Phase 1.1 เดิมยังตรวจว่า Catalog ตอบเพียง 6 fields

## Open decisions ก่อนเปลี่ยน Public API

- payload/result ของ `/builds/validate` ยืนยันแล้ว และ HTTP statuses อธิบายใน [BUILD_API.md](BUILD_API.md)
- วางแผนเพิ่ม `specs` และ RAM `ram_model_id` ลง product response; รอบนี้ **ยังไม่เพิ่ม**
- ยืนยันชุด category ทั้งหมด รวมถึง normalization ของ socket/RAM type identifiers
  ตอนนี้ repository trim whitespace แต่เทียบค่าตรงตัว ไม่แปลงตัวพิมพ์เอง
- SKU grouping/ข้อมูล RAM model เพิ่มเติม และ stock limits ยังไม่อยู่ใน scope
- เกณฑ์ความครบ หมวดบังคับ และกรณีกราฟิกออนบอร์ดยังไม่ตกลง: **ไม่เปิด public publication**
- พฤติกรรมกับสินค้าที่ deactivate หลังบันทึก build โดยเฉพาะของที่มีอยู่แล้ว ยังต้องตกลง
  ตอนนี้ internal service ปฏิเสธ inactive non-RAM แบบ fail-closed; owned RAM ใช้ model type ได้
- HTTP validation ปฏิเสธ duplicate product/model entries ตามกฎภายในที่ระบุใน API contract
- การจำกัด CPU/motherboard ต่อ build ยังไม่ตกลง ภายในเทียบทุกคู่ที่เลือกโดยไม่เดาจำนวนสูงสุด
- Supabase adapter, migration integration tests, final RLS และ Auth/owner authorization ยังไม่ทำ
- การบันทึก private draft เป็น API/persistence จะทำแยกหลังตกลง Auth/build schema; รอบนี้มีการประเมิน draft ภายใน
