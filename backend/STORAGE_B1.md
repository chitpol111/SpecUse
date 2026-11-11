# B1 — SATA-first สำหรับ synthetic fixture เท่านั้น

เพิ่มเฉพาะการตรวจ direct internal SATA ตามขอบเขตที่ Fluke ยืนยัน หลังช่วง A
ใช้ API/schema เดิมและ opt-in `validation_mode=gpu_storage_v1`; default legacy response ไม่เปลี่ยน
branch นี้เริ่มจาก main `2abf3943de5a2a574483eb2c821ba2e12da54c4e`
ยังไม่เพิ่มสินค้า/หลักฐานผู้ผลิตจริง ไม่เชื่อม Supabase/Auth/Admin และไม่มี migration ใหม่

## รัน Backend

ต้องมี Git, Go 1.22+ และ curl จาก root ของ checkout branch feat/backend-storage-b1:

```bash
cd backend
APP_ENV=local \
LOCAL_CATALOG_FILE=./testdata/catalog.storage-b1.synthetic.local.json \
PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

Base URL `http://127.0.0.1:8080/api/v1`; React port 3000 ใช้ได้ทั้ง localhost และ 127.0.0.1
เปลี่ยน PORT แล้วต้องเปลี่ยน API base URL ตาม CORS ต้องตรง origin ไม่มี slash ท้าย
คำสั่งใช้ Bash; PowerShell ต้องตั้ง environment ตามรูปแบบของ PowerShell
ไม่ต้องใช้ token/cookies/Supabase/secrets หรือรัน SQL; โปรแกรมไม่อ่าน .env อัตโนมัติ

fixture ใหม่มี 28 products / active 26: ข้อมูลช่วง A เดิม 21 รายการและข้อมูลจำลอง B1 เพิ่ม 7 รายการ
SKU ใหม่ชื่อขึ้นต้น [SYNTHETIC FIXTURE ONLY], UUID 940...; CPU/SATA/ตัวอย่าง GPU/M.2 ใช้ 910... เดิม
source synthetic-v1 และ URL .invalid เป็นข้อมูลแต่งเพื่อทดสอบ ไม่ใช่เอกสารผู้ผลิต
ไม่มี ASUS PRIME B550-PLUS หรือ Kingston SA400S37/480G เป็นสินค้า/หลักฐานใน fixture
Source ของบอร์ด sharing จำลองเป็น rule ใน test model เท่านั้น ไม่อ้างว่าฮาร์ดแวร์จริงมีเงื่อนไขนี้
fixture โหลดได้เฉพาะ APP_ENV=local พร้อม path และ bind เฉพาะ loopback; ค่าเริ่มต้นยัง catalog ว่าง
constructor ปกติยังปฏิเสธ synthetic provenance ไม่มีการ embed ข้อมูลลง binary
fixtures เดิมไม่เปลี่ยน: catalog.local.json และ catalog.gpu-storage.synthetic.local.json

## B1 ตรวจอะไร

- ต้องมี CPU และ motherboard อย่างละหนึ่ง (นับ quantity รวม owned) และเลือก CPU profile ได้เพียงชุดเดียวจาก UUID
  ขาด/หลายตัว/หลาย profile/มี profile ที่อาจใช้ได้แต่ applicability ไม่ทราบ ให้ unknown ไม่เพิ่ม HTTP limit ใหม่
- Storage ต้องเป็น connection_type=sata, connector=sata_data และ form_factor=2_5_inch หรือ 3_5_inch
  ขนาดตัว drive สำหรับใส่ case ไม่อยู่ในการตรวจนี้; display/capacity/power ไม่ถูกใช้รับรอง motherboard
- ตรวจ evidence coverage ของ /host/connection_type, /host/connector, /host/form_factor ของทุก SKU
  และ /connectivity_profiles ของ motherboard ซึ่งรวมความครบของรายการ ports/constraints
  evidence path ตรงหรือ ancestor ครอบคลุมได้ เช่น /host; ต้องอ้าง source ID ที่มีจริง
  มี sources/evidence ที่ไม่ครอบคลุม claims เหล่านี้อย่างเดียวไม่พอให้ผ่าน
- profile ต้องยืนยันว่าไม่ต้องรอ firmware context และ resource_rules=[] ที่มีหลักฐานรองรับ
  null ไม่แปลว่าไม่มี rule; B1 ยังไม่ประเมิน nonempty rule set ทุกชนิดอย่างอนุรักษ์นิยม
- ใช้เฉพาะพอร์ต connector=sata_data และ supported_connection_types มี sata
  พอร์ต M.2 ที่ไม่ได้ใช้อยู่ไม่ถูกนับเป็น direct SATA; ไม่อนุมาน adapter
  พอร์ต connector/protocol ที่ไม่ทราบหรือ class ที่อาจเปลี่ยนคำตอบแต่ยังไม่รองรับให้ unknown
- จัดทุก instance ตาม quantity ลงพอร์ตต่างกันใน pool เดียวกันทุก SKU รวมของ owned
  กฎคงที่: เรียง product UUID แล้ว instance 1..quantity และเรียง port_id; allocation เป็นข้อมูลภายใน ไม่เพิ่ม JSON fields
  ราคาใช้ (quantity-owned_quantity)*catalog.price_thb ตามเดิม; owned ไม่ได้เพิ่ม/ลดจำนวนช่องที่ต้องใช้

การผ่านเป็นผลใน synthetic test model เท่านั้น ไม่ใช่การยืนยันสเปกสินค้าจริง
การตรวจ JSON/source reference/path coverage ไม่ใช่การตรวจรับรองความถูกต้องของข้อความในเอกสารผู้ผลิต
ก่อนใช้กับสินค้าจริงต้องอนุมัติ CPU/SKU/hardware revision และหลักฐานตรงเงื่อนไขแยกต่างหาก

## passed / failed / unknown

STORAGE_MOTHERBOARD และ MOTHERBOARD_RESOURCES เป็น aggregate ของรายการ Storage ที่เลือก:

| เงื่อนไข | Check ทั้งสอง |
|---|---|
| ข้อมูล/profile/evidence/constraints ที่จำเป็นครบ และจัด SATA ทุกลูกลงคนละพอร์ตได้ | passed |
| ข้อมูลครบแต่ไม่มี direct SATA port ที่รองรับ หรือจำนวนพอร์ตที่ใช้ได้ไม่พอ | failed |
| ports=null, evidence ไม่พอ, profile เลือกไม่ได้ หรือข้อมูลจำเป็นยังขาด | unknown |
| เลือก GPU/M.2, มี nonempty sharing rules หรือ BIOS/context ที่ยังไม่ประเมิน | unknown |

กรณีพอร์ตไม่พอ ทั้งสอง checks เป็น failed ไม่รายงานว่า Storage aggregate ผ่านทั้งที่ยังจัดครบไม่ได้
ports=[] ที่มี evidence ยืนยันความครบและไม่มี constraints ค้าง ทำให้ failed ต่างจาก ports=null ที่ unknown
ถ้าไม่มี evidence ยืนยันแม้ ports=[] ก็ยัง unknown ไม่ใช้การค้นพบเพียงบางส่วนพิสูจน์ failure
ถ้ามี GPU/M.2 หรือ constraints ที่ B1 ยังไม่ประเมิน จะยังไม่พยายามตัดสิน SATA allocation จากพอร์ตดิบ
USB/SAS/other ที่ contract ยังไม่รองรับเป็น unknown/UNSUPPORTED_DEVICE_CLASS และยังนับ software limit

CPU/RAM checks เดิมยังทำงาน: proven failed ชนะ unknown เสมอ
SATA ผ่าน แต่ required PSU/Case checks ยัง unknown → overall unknown และ opt-in compatible=false
SATA failed หรือ CPU/RAM conflict → overall failed แม้ PSU/Case unknown
ไม่มี overall passed ใน B1; build_completeness ยัง not_evaluated และไม่เปิด publish
legacy compatible ยังคงความหมาย CPU/RAM เดิม ไม่มี compatibility object และตัวอย่างเดิมยัง 11500

Check/FieldReference schema ไม่เปลี่ยน ทุก check ส่ง missing_fields และ unsupported_fields เป็น arrays เสมอ
missing ใช้ path ของข้อมูล/coverage ที่ขาด เช่น /specs/evidence พร้อม product_id/category ของสินค้านั้น
evidence array ที่มีแล้วอาจถูกอ้างใน missing_fields เมื่อยังขาดหลักฐานที่ครอบคลุม claim ที่ต้องใช้
sharing rules ที่ทราบว่ามีแต่ยังไม่ประเมินอยู่ unsupported_fields ไม่ปลอมเป็น missing rule
BIOS ที่ยังไม่ทราบใช้ location=context, path=/bios_version พร้อม product_id/category ของ motherboard
ไม่มีการอนุญาตให้ client ส่ง BIOS, port choice, ราคา/specs/evidence หรือผลตรวจใน body

## ทดลอง API และ JSON ตัวอย่าง

GET list/detail ใช้ snapshot เดียวกับ validate และยัง active-only:

```bash
curl -i 'http://127.0.0.1:8080/api/v1/products?category=storage'
curl -i 'http://127.0.0.1:8080/api/v1/products/94000000-0000-4000-8000-000000000002'

# quantity=3,owned=2 ใช้ 3 พอร์ต ซื้อเพิ่ม 1 ลูก: SATA checks passed, overall unknown, total=1000
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1' \
  -H 'Content-Type: application/json' --data-binary @examples/storage-b1/validate.request.json

# ต้องการ 4 ลูกบนบอร์ด 3 พอร์ต: HTTP 200/failed, ซื้อเพิ่ม 1 ลูก total=1000
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1' \
  -H 'Content-Type: application/json' --data-binary @examples/storage-b1/validate-too-many-ports.request.json

# Legacy: compatible=true และ total=11500 ตามเดิม
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Content-Type: application/json' --data-binary @examples/validate.request.json
```

body คง items:[{product_id,quantity,owned_quantity}], ram_items ตามเดิม ไม่เพิ่ม fields
แต่ละ prefix ใน examples/storage-b1 มี .request.json และ .response.json เต็มที่ tests เทียบกับ API:

| prefix | ผล SATA checks / overall / ยอดซื้อเพิ่ม |
|---|---|
| validate | passed / unknown / 1000 |
| validate-multi-sku | passed / unknown / 2500; SKU สองตัวใช้ pool พอร์ตเดียวกัน |
| validate-too-many-ports | failed / failed / 1000 |
| validate-no-ports | failed / failed / 1000; ยืนยัน ports=[] |
| validate-null-ports | unknown / unknown / 1000 |
| validate-sharing | unknown / unknown / 1000 |
| validate-bios | unknown / unknown / 1000 |
| validate-evidence-missing | unknown / unknown / 1000 |
| validate-gpu | unknown / unknown / 1000 |
| validate-m2 | unknown / unknown / 1000 |

ดู [Storage list](examples/storage-b1/products-storage.response.json),
[motherboard detail](examples/storage-b1/product-motherboard.response.json),
[passed checks แต่ overall unknown](examples/storage-b1/validate.response.json),
[พอร์ตไม่พอ](examples/storage-b1/validate-too-many-ports.response.json),
[ports ไม่ทราบ](examples/storage-b1/validate-null-ports.response.json)
ตัวอย่าง POST ในโฟลเดอร์ gpu-storage-a อัปเดตผลที่ runtime ใหม่ส่งจริง; requests/fixtures เดิมไม่เปลี่ยน

HTTP/status/error envelope เดิม: hardware failed/unknown เป็น 200;
400 VALIDATION_ERROR/UNSUPPORTED_VALIDATION_MODE/UNSUPPORTED_BUILD_SCOPE, 404 NOT_FOUND,
405 METHOD_NOT_ALLOWED, 413 PAYLOAD_TOO_LARGE, 415 UNSUPPORTED_MEDIA_TYPE, 500 INTERNAL_ERROR
proven issues ใช้ STORAGE_INTERFACE_MISMATCH หรือ MOTHERBOARD_RESOURCE_CONFLICT ตาม contract
GPU รวม<=1, Storage ทุกชนิดรวม<=8 เป็น software limits เดิม รวม owned โดยไม่บวกซ้ำ
สินค้าที่ไม่พบ/inactive ยัง 404 ก่อน limit; RAM selection และ integer-overflow checks เดิมไม่เปลี่ยน

## Tests

```bash
go test ./...
go vet ./...
go test -race ./internal/...
```

ไม่ใช้ Supabase/secrets; lifecycle tests เปิด loopback ชั่วคราว
ครอบคลุม owned/หลาย SKU/พอร์ตคนละช่อง/order independence/N และ N+1,
null vs []/evidence coverage/profile/CPU/GPU/M.2/sharing/BIOS, failure priority,
legacy 11500/ราคา/limits/404/ห้าม client เลือกพอร์ตหรือส่ง BIOS และ CORS ทั้งสอง origin
tests เปรียบเทียบผล compatibility/ราคาโดยไม่บังคับเรียง priced items ใหม่ ซึ่งยังเรียงตาม request แบบเดิม

ยังไม่ทำ GPU/M.2 allocation, sharing-rule evaluation, BIOS execution/context input,
PSU/Case/boot/RAID/adapters/ความครบของ build หรือ manufacturer SKU ingestion
