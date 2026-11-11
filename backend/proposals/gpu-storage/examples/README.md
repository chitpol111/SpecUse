# คู่มือ JSON สำหรับ Fluke review PR #3

ทั้งหมดเป็น **ข้อเสนอ ยังไม่ใช่ API ที่ server รองรับ** ไม่มีการแก้ runtime fixture หรือโค้ด
JSON body ไม่มี comments หรือ field `synthetic` ที่เพิ่มนอก contract; แยก dataset ด้วยชื่อไฟล์/UUID/ชื่อสินค้า
GET ใช้ envelope เดิม และ Product ใน detail ต้องเท่ากับ Product ID เดียวกันใน list ของ dataset นั้น

## Dataset และขอบเขตการใช้

| ชุด | Catalog | ความหมาย |
|---|---|---|
| Schema เดิม | [catalog-list.proposed.json](catalog-list.proposed.json) | GPU/Storage hardware fields เป็น null; ราคา 0 เป็น placeholder; CPU/board ใช้ local fixture เดิม |
| Synthetic ใหม่ | [catalog.synthetic.proposed.json](catalog.synthetic.proposed.json) | metadata มีค่าเพื่อทดสอบ schema/logic เท่านั้น; ไม่ใช่สินค้า/ราคาจริง ไม่ใช่ seed ที่ server โหลด |

Synthetic ใช้ UUID `91000000-0000-4000-8000-...` และ RAM model `92000000-0000-4000-8000-...`
ทุกชื่อมี `[SYNTHETIC FIXTURE ONLY]`; `synthetic-v1`/URL `.invalid` ไม่ใช่ manufacturer evidence
ค่าทั้งหมดเป็น input ของ artificial test model ไม่ได้นำ S1–S7 มาอ้างรับรองสินค้าจำลอง
production ห้ามรับ synthetic source เป็นหลักฐาน; client ส่ง specs/ราคา/evidence มาแทน catalog ไม่ได้

| ID ท้าย UUID สินค้า | หมวด/บทบาท | สมมติฐานทดสอบ |
|---|---|---|
| 001 | CPU A | socket SYNTH_SOCKET_A |
| 002 | Board A | profile ตรง CPU 001; PCIE_1 สำหรับ GPU, M2_1 NVMe, SATA_1–3; ไม่มี sharing rule ใน test model |
| 003 | RAM SKU | DDR5, 1 module/package; model 920...001 |
| 004 | GPU A | physical PCIe x16, ยืนยัน synthetic mode gen4/x16 |
| 005 | Storage NVMe | M.2 key M, 22×80 mm, synthetic mode gen4/x4 |
| 006 | Storage SATA | 2.5 นิ้ว SATA; M.2/PCIe fields=null เพราะไม่ applicable ไม่ใช่ข้อมูลขาด |
| 007 | Board B | profile ตรง CPU; PCIE_1 และ SATA_1–3 เท่านั้น ไม่มี M.2/ทางเลือก NVMe ใน test model |
| 008 | GPU B | เหมือนรูปแบบ GPU A แต่ connector=null โดยเจตนาเพื่อทดสอบ missing metadata |
| 009 | Storage USB | ทราบ connection_type=usb แล้ว แต่ phase B ไม่รองรับ checker นี้; ยังนับ Storage limit |
| 010 / 011 | PSU / Case | 6-field placeholders สำหรับ request อนาคต C เท่านั้น ไม่ได้ออกแบบ typed specs หรือรับรองความเข้ากันได้ |

สำหรับ Board A ชุดนี้เท่านั้น: GPU 004→PCIE_1, NVMe 005→M2_1, SATA 006 quantity=2→SATA_1/SATA_2
CPU UUID/profile/ports/resource_rules ครบใน artificial model จึงผ่าน motherboard checks ได้ในตัวอย่างช่วง B
ไม่ใช่กฎว่า motherboard จริงทุกตัวไม่มี sharing หรือ CPU socket ตรงแล้วใช้ทุก feature ได้

## GET list/detail

- ชุดเดิม: [GPU](gpu-detail.proposed.json), [Storage](storage-detail.proposed.json), [motherboard](motherboard-detail.proposed.json)
- ชุด synthetic: [GPU](gpu-detail.synthetic.proposed.json), [Storage](storage-detail.synthetic.proposed.json),
  [motherboard](motherboard-detail.synthetic.proposed.json); ID/specs/ราคาเท่ากับ list synthetic
- CPU/RAM/Board B/SATA/USB/GPU B และ future placeholders อ่านจาก list synthetic ได้;
  detail ของ ID เหล่านี้จะห่อ Product เดียวกันด้วย `{"data":{"product":...}}` ไม่ต้องมี mapping แยกใน FE

## POST validate: full responses และ request คู่กัน

ทุก response ด้านล่างเป็น HTTP 200 ตัวอย่างของ opt-in
`POST /api/v1/builds/validate?validation_mode=gpu_storage_v1`
ใช้ `Content-Type: application/json`; request มีเพียง items/ram_items ตามสัญญาเดิม
ไม่ส่ง filename/dataset/scenario หรือผลที่คาดหวังเข้า API

| กรณี | Request | Full response | Overall / bool / ราคา |
|---|---|---|---|
| null metadata ชุดเดิม | [request](validate.request.proposed.json) | [response](validate-unknown.response.proposed.json) | unknown / false / 0 placeholder |
| B: checks motherboard ผ่าน | [request](validate.synthetic.request.proposed.json) | [response](validate-motherboard-passed.response.proposed.json) | unknown / false / 8000 |
| B: NVMe บน Board B SATA-only | [request](validate-failed.request.proposed.json) | [response](validate-failed.response.proposed.json) | failed / false / 7500 |
| B: GPU B connector ขาด | [request](validate-metadata-missing.request.proposed.json) | [response](validate-metadata-missing.response.proposed.json) | unknown / false / 8000 |
| B: USB known unsupported | [request](validate-unsupported.request.proposed.json) | [response](validate-unsupported.response.proposed.json) | unknown / false / 6500 |
| C: all passed แบบมีเงื่อนไขอนาคต | [request](validate-passed.future.request.proposed.json) | [response](validate-passed.future.response.proposed.json) | passed / true / 11500 |

**Partial passed ไม่ใช่ overall passed:** ชุด B motherboard-passed มี CPU_SOCKET, RAM_TYPE,
GPU_MOTHERBOARD, STORAGE_MOTHERBOARD และ MOTHERBOARD_RESOURCES ผ่าน แต่ PSU/Case checks ยัง unknown
จึง overall unknown; ชุด failed พบข้อขัดแย้ง NVMe แน่นอนจึง overall failed แม้ PSU/Case ยัง unknown

ราคา synthetic B ปกติ: CPU owned 1 จ่าย 0 + board 4000 + GPU owned 1 จ่าย 0 + NVMe 2000
บวก SATA quantity 2/owned 1 จ่าย 1000 + RAM ต้องการ 2/owned 1 ซื้อ single package 1000 = 8000
SATA ใช้ 2 ports แม้ซื้อเพิ่มลูกเดียว; RAM modules_per_package และ ram_model_id มาจาก SKU ใน list
Board B ถูกกว่า 500 → 7500; เปลี่ยน NVMe 2000 เป็น USB 500 → 6500; request อนาคตเพิ่ม PSU 2000+Case 1500 → 11500

**ขอบเขตตัวอย่าง C:** full passed เป็นตัวอย่างรูปแบบ/UI ภายใต้ precondition ว่าอนาคตตกลง PSU/Case/system
metadata+installation context และมี checker ที่ตรวจครบจนทุก required check ผ่านแล้วเท่านั้น
Catalog นี้มีเพียง PSU/Case placeholders จึง **คำนวณ passed ตัวอย่างนี้จาก Catalog นี้ไม่ได้**
ไม่อ้างว่าอุปกรณ์เหล่านั้น fit/จ่ายไฟพอจริง ไม่ใส่ PSU/Case fields ที่ยังไม่ตกลง และไม่ใช่ผลที่ช่วง A/B ส่งได้
ก่อนนำไปเป็น executable acceptance test ต้องเพิ่ม authoritative C dataset และยืนยัน contract แยก

Default `/builds/validate` ยังอ่าน bool ตาม legacy CPU/RAM และไม่เพิ่ม compatibility object;
FE ที่ opt-in ต้องตรวจ mode echo และอ่าน status/coverage ไม่ใช้ bool=false แทนคำว่า incompatible ทุกกรณี
ห้ามเปลี่ยน default client โดยไม่ตกลง และห้ามใช้ all-passed ตัวอย่าง C ยืนยันความครบ/publish

## missing_fields / unsupported_fields

Check ทุกสถานะมีทั้งสอง arrays ชนิด `FieldReference[]`; ไม่มีรายการส่ง [] ห้ามส่ง null/string[]
FieldReference ทุกตัวมี location/path/product_id/category เสมอ ตามหัวข้อ 5.1 ใน [proposal](../README.md)
JSON Pointer ใช้ root ตาม location ไม่ใช่ path ย่อที่ FE ต้องเดาว่าอ้าง Product ใด

```json
{
  "location": "catalog",
  "path": "/specs/host/connector",
  "product_id": "91000000-0000-4000-8000-000000000008",
  "category": "gpu"
}
```

GPU B ขาด connector: ดู GPU_MOTHERBOARD ใน [metadata-missing response](validate-metadata-missing.response.proposed.json)
ที่มี object นี้ใน missing_fields และ unsupported_fields=[]

ยังไม่เลือกอุปกรณ์ใช้ product_id=null พร้อมหมวดที่ขาด ไม่ใช้ UUID สมมติ:

```json
{"location":"request","path":"/items","product_id":null,"category":"psu"}
```

ดู GPU_PSU/SYSTEM_PSU ใน [motherboard-passed response](validate-motherboard-passed.response.proposed.json)
ส่วนบริบททั้งระบบที่ยังไม่มีใช้ null/null โดยไม่เพิ่ม field ใหม่ใน request:

```json
{"location":"context","path":"/installation_context","product_id":null,"category":null}
```

ดู SYSTEM_CASE ใน [null-metadata response](validate-unknown.response.proposed.json)
Storage USB **มี** connection_type=usb แล้ว จึงไม่ใส่ field นี้ใน missing_fields:
ดู STORAGE_MOTHERBOARD ใน [unsupported response](validate-unsupported.response.proposed.json)
ที่ missing_fields=[] และ unsupported_fields มี catalog reference ของ Storage 009
ส่วน check ที่ผ่าน/ล้มเหลวจากข้อมูลครบดู [failed response](validate-failed.response.proposed.json): ทั้งสอง arrays=[]
NOT_IMPLEMENTED ล้วน ๆ ก็ใช้ทั้งสอง arrays=[] ได้เมื่อข้อมูลครบ เพราะการยังไม่มี checker ไม่ใช่ metadata ขาด
ถ้ามีข้อมูลขาดและชนิดที่ยังไม่รองรับพร้อมกันให้รายงานทั้งสองรายการตามจริง

## ตารางสถานะ/จำนวนและ error

- [overall-status.scenarios.proposed.json](overall-status.scenarios.proposed.json): failed ชนะ unknown;
  ไม่มี failed แต่มี required unknown → unknown; ทุก required ผ่านและไม่ว่าง → passed; [] → unknown
- [storage-counting.scenarios.proposed.json](storage-counting.scenarios.proposed.json): 6 กรณี quantity/owned/mixed class,
  software limit 8 รวม USB/ของ owned, มากกว่า 8 → 400 UNSUPPORTED_BUILD_SCOPE และ unknown product → 404 NOT_FOUND ก่อน limit
- [check-outcomes.proposed.json](check-outcomes.proposed.json): check fragments passed/failed/unknown
  ภายใต้ preconditions ที่ระบุ ไม่ใช่ full response หรือผลจาก null metadata

Files ที่ลงท้าย scenarios เป็นเอกสาร test cases ไม่ใช่รูปแบบ response envelope ใหม่;
expected_error ใช้ envelope เดิม `{"error":{"code":"...","message":"..."}}`
การผ่าน software limit ไม่แปลว่าฮาร์ดแวร์ติดตั้งได้ 8 ลูก และไม่ใช่ผล compatibility ผ่าน
HTTP/validation/method/media/size/repository error อื่นคงตามตารางใน proposal หลัก

## เกณฑ์ตรวจเอกสารรอบนี้

JSON ทุกไฟล์ต้อง parse ได้; list/detail ตรงกันตาม dataset; UUID/category/ราคา/RAM selection ตรงกับ request
และ Catalog; totals คิดเฉพาะซื้อเพิ่ม; check keys/FieldReference รูปแบบเดียวกัน; aggregate/unverified_domains
สอดคล้องทุก response; known unsupported ไม่ถูกใส่เป็น missing และ owned ไม่ถูกนับซ้ำใน limit
ตรวจเฉพาะเอกสาร ไม่ใช่การรัน implementation หรือทดสอบว่า API ปัจจุบันรองรับ proposal นี้แล้ว

ตรวจ parse ตัวอย่างทั้ง 23 ไฟล์ซ้ำได้จากโฟลเดอร์ backend:

```bash
python3 - <<'PY'
import json
from pathlib import Path
files = sorted(Path('proposals/gpu-storage/examples').glob('*.json'))
for file in files:
    json.loads(file.read_text())
print(f'Parsed {len(files)} JSON documents')
PY
```
