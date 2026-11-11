# GPU/Storage ช่วง A — Catalog + ราคา

เอกสารนี้บันทึกขอบเขตช่วง A; runtime ล่าสุดเพิ่ม [B1 SATA-first](STORAGE_B1.md) ใน opt-in mode เดิมแล้ว
direct SATA ที่ผ่านเงื่อนไข B1 ตรวจ allocation ได้ แต่ GPU/M.2/PSU/Case ยัง unknown ตาม coverage
ตัวอย่างเดิมที่มี GPU/M.2 ยังคง unknown และ fixtures/ราคาเดิมไม่เปลี่ยน

Implement เฉพาะช่วง A ตาม [PR #3](https://github.com/chitpol111/SpecUse/pull/3)
และ [contract commit 3581447](https://github.com/chitpol111/SpecUse/tree/358144723d71941016956c8e6f760b15f683eba7/backend/proposals/gpu-storage)
จาก main `642bdab327adfd9761f9debfa05b52fd196df9fa` โดยไม่ merge branch เอกสารเข้ามา
ใช้ net/http/JSON และ In-memory Repository ยังไม่เชื่อม Supabase/Auth/Admin

## รันให้ฟลุคทดลอง

ติดตั้ง Git, Go 1.22+ และ curl จาก root ของ checkout `feat/backend-gpu-storage-a`:

```bash
cd backend
APP_ENV=local \
LOCAL_CATALOG_FILE=./testdata/catalog.gpu-storage.synthetic.local.json \
PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

Base URL `http://127.0.0.1:8080/api/v1`, API port 8080, React port 3000
ถ้าเปลี่ยน PORT ให้เปลี่ยน API base URL ตาม; CORS ต้องตรง origin จริง ไม่มี slash ท้าย
ไม่ต้องใช้ token/cookies/secret/database หรือรัน SQL migrations; ไม่อ่าน `.env` อัตโนมัติ
ตัวอย่างใช้ Bash; PowerShell ให้ตั้ง environment ด้วยรูปแบบของ PowerShell

Fixture ใหม่มีข้อมูลเดิม 9 รายการบวก synthetic 12 รายการ รวม 21/active 19
ไฟล์ `testdata/catalog.local.json` เดิมไม่เปลี่ยน จึงยังทดลองยอด 11500 ได้ทั้งสอง fixture
ข้อมูลใหม่มีชื่อ `[SYNTHETIC FIXTURE ONLY]`, UUID 910.../930..., source `synthetic-v1` และ URL `.invalid`
เป็นข้อมูลทดสอบ ไม่ใช่สเปก/ราคาจริงหรือ manufacturer evidence ไม่ถูก embed ใน binary
ต้องระบุ APP_ENV=local พร้อม path จึงโหลดได้ และ bind เฉพาะ loopback
ไม่ระบุ path จะได้ repository ว่าง; non-local ระบุ path หยุดก่อนอ่านไฟล์
constructor ปกติปฏิเสธ synthetic provenance; local loader ใช้ NewLocalBuildMemory หลังผ่าน environment gate

## GET list/detail และ metadata

คง 6 fields `id,name,category,brand,price_thb,is_active`, UUID, category lowercase และราคาเงินบาทจำนวนเต็ม
active-only, sort/filter/pagination/envelopes เดิม; list/detail และ validate ใช้ immutable snapshot เดียวกัน
deep-copy nested pointers/slices ที่ขอบเขต repository ไม่ให้ caller แก้ metadata/ราคาของคำขออื่น

```bash
curl -i 'http://127.0.0.1:8080/api/v1/products?category=gpu'
curl -i 'http://127.0.0.1:8080/api/v1/products?category=storage'
curl -i 'http://127.0.0.1:8080/api/v1/products/91000000-0000-4000-8000-000000000004'
curl -i 'http://127.0.0.1:8080/api/v1/products/91000000-0000-4000-8000-000000000002'
```

JSON ที่เทียบกับ API ใน tests: [GPU list](examples/gpu-storage-a/products-gpu.response.json),
[Storage list](examples/gpu-storage-a/products-storage.response.json),
[motherboard detail](examples/gpu-storage-a/product-motherboard.response.json)
ID เดียวกันใน list/detail มี Product/specs/ราคาตรงกัน ไม่ต้อง hardcode mapping ใน FE

fields ใหม่ใช้ null เมื่อไม่ทราบ รวม object/array; [] คือ reviewed empty ไม่ใช่ unknown
เก็บค่า 0/false ที่มีความหมายตามจริง ไม่แทน null และไม่ใช้ omitempty กับ nullable hardware fields
CPU/RAM fields เดิมและ RAM ram_model_id ระดับเดียวกับ id ไม่เปลี่ยน

| Category / specs | Fields หลัก | ชนิด/หน่วย |
|---|---|---|
| gpu.display | chipset_name, vram_capacity_bytes | string?, integer? bytes; แสดงผลเท่านั้น |
| gpu.host | connector, supported_link_modes | pcie_x16/other?, LinkMode[]? |
| gpu.power | board_power_w, min_system_power_w, connector_options | number? W, integer? W, PowerOption[]? |
| gpu.dimensions | length_mm, height_mm, thickness_mm, occupied_slots | number? mm, integer? slots |
| storage.display | media_type, capacity_bytes, sequential_read_mb_s, sequential_write_mb_s | ssd/hdd/other?, integer? bytes, number? decimal MB/s |
| storage.host | connection_type, connector, form_factor, m2_keying, width_mm, length_mm, supported_link_modes | enums?, integer? mm, LinkMode[]? ตาม contract |
| storage.power | connector_type, max_operating_power_w, startup_power_w | enum?, number? W |
| storage.dimensions | width_mm, length_mm, height_mm | number? mm |
| motherboard | connectivity_profiles, sources, evidence | IOProfile[]?, Source[]?, Evidence[]? เพิ่มจาก cpu_socket/ram_type |
| gpu/storage | sources, evidence | Source[]?, Evidence[]? |

LinkMode `{pcie_generation:integer>=1,lanes:1|2|4|8|16}`; IOProfile/slots/ports/resource rules,
PowerOption และ provenance มี typed structs ตาม contract ใน `internal/hardware/metadata.go`
ช่วง A ตรวจ stored shape/enums/ranges/source references เท่านั้น **ไม่ประเมิน hardware rules**
ไม่ดึงสเปกจากชื่อ/VRAM/ความจุ/ราคา หรือเติมข้อมูลสินค้าจริงเอง
GPU/Storage ที่ metadata ขาดยังแสดง/คิดราคาได้ โดย specs sections เป็น null
motherboard เดิมเพิ่ม connectivity_profiles/sources/evidence=null แบบ additive; PSU/Case/cooler ยังไม่มี typed specs

## Validate legacy และ opt-in

Default `POST /builds/validate` คง JSON/CPU+RAM semantics และ scope เดิม ไม่มี compatibility object
legacy compatible=true ไม่ใช่ผลตรวจ GPU/Storage/PSU/Case ไม่ใช่เกณฑ์ build ครบ/publish

```bash
# Legacy: compatible=true, total_price_thb=11500
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Content-Type: application/json' --data-binary @examples/validate.request.json

# Opt-in A: total=8000, overall unknown, compatible=false
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1' \
  -H 'Content-Type: application/json' --data-binary @examples/gpu-storage-a/validate.request.json

# CPU/RAM conflicts: failed แม้ PSU/Case unknown
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1' \
  -H 'Content-Type: application/json' --data-binary @examples/gpu-storage-a/validate-cpu-conflict.request.json
curl -i 'http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1' \
  -H 'Content-Type: application/json' --data-binary @examples/gpu-storage-a/validate-ram-conflict.request.json
```

body เดิมเท่านั้น: `items:[{product_id,quantity,owned_quantity}]` และ
`ram_items:[{ram_model_id,required_modules,owned_modules}]`; ต้องมีทั้งสอง arrays ใช้ [] ได้
GPU/Storage ใช้ items เดิม ไม่ส่ง specs/ราคา/evidence/mode/ผลตรวจใน body
ราคา non-RAM = `(quantity-owned_quantity)*catalog.price_thb`; RAM ใช้ active package selection เดิม
เลือกโมดูลพอดีก่อน แล้วส่วนเกินน้อยที่สุด แล้วราคาต่ำสุดและ deterministic tie-break เดิม

### Software limits เฉพาะ opt-in

GPU SUM(quantity)<=1; Storage SUM(quantity)<=8 จาก category ใน catalog ของ Backend
นับ Storage **ทุกชนิด** รวม USB/SAS/other/metadata ขาด/owned; owned เป็นส่วนหนึ่งของ quantity
ไม่บวกซ้ำหรือหักจาก limit เช่น quantity=3,owned=2 นับติดตั้ง 3 และคิดเงินเพิ่ม 1
8 ลูกไม่รับรองว่าฮาร์ดแวร์รองรับ 8 ลูก; ports/power/พื้นที่เป็นงาน B/C
ตรวจรูปแบบ/duplicates → resolve ทุก product/model/active package ที่ต้องซื้อ → limit → ราคา/checks
สินค้าที่ไม่พบ/inactive แม้ owned ยัง 404 ก่อน limit; duplicate IDs ตามเดิมยังเป็น 400
legacy ไม่ถูกเพิ่ม limits ใหม่เงียบ ๆ

### Response โหมดใหม่

คง data เดิมและเพิ่ม `compatibility={mode,status,checks,warnings,unverified_domains,build_completeness}`
mode ต้อง echo `gpu_storage_v1`; FE ห้ามอ่าน legacy bool เป็นผลใหม่เมื่อ compatibility/mode หาย
`compatible = (compatibility.status=="passed")` เฉพาะ opt-in; false ไม่ได้แปลว่า incompatible เสมอ

- มี check failed ที่ยืนยันได้ → overall failed แม้มี unknown
- ไม่มี failed แต่มี required unknown หรือไม่มี checks → overall unknown
- passed ได้เมื่อ required checks ไม่ว่างและผ่านทั้งหมดเท่านั้น
- A ตรวจ CPU_SOCKET/RAM_TYPE ที่เกี่ยวข้องได้; GPU/STORAGE_MOTHERBOARD, MOTHERBOARD_RESOURCES,
  GPU/STORAGE_PSU/CASE และ SYSTEM_PSU/CASE ยัง unknown แม้ synthetic metadata ครบ
- เลือกอุปกรณ์ใดต้องมี SYSTEM_PSU/SYSTEM_CASE; เลือก GPU/Storage ต้องส่ง checks ของหมวดนั้นแม้ขาด motherboard/CPU
- A ยังไม่มี overall passed แต่มี overall failed จาก CPU/RAM conflict ได้
- issues ใช้ `{code,message}` เฉพาะ conflict ที่ยืนยันได้; unknown ไม่ใช่ server error
- unverified_domains รวม scopes ที่ unknown แม้ overall failed; build_completeness=`not_evaluated` เสมอ

ทุก Check มี `{code,scope,status,reason_code,message,product_ids,missing_fields,unsupported_fields}`
สอง arrays หลังเป็น FieldReference[] เสมอ ไม่มีรายการส่ง []:

```json
{"location":"catalog","path":"/specs/host/connector","product_id":"91000000-0000-4000-8000-000000000008","category":"gpu"}
```

catalog path เป็น JSON Pointer จาก Product พร้อม ID/category จริงของสินค้าที่เกี่ยวข้อง
ยังไม่เลือก PSU ใช้ `{location:"request",path:"/items",product_id:null,category:"psu"}`
context ระบบใช้ `{location:"context",path:"/installation_context",product_id:null,category:null}`
context paths บอกสิ่งที่ขาด ไม่ใช่ body fields ใหม่ที่ client ส่งได้
known USB อยู่ใน unsupported_fields ไม่ใช่ missing connection_type; M.2 fields ของ SATA ไม่ applicable จึงไม่เป็น missing
NOT_IMPLEMENTED ไม่ใช่ข้อมูลขาด เช่น GPU_MOTHERBOARD ของชุด metadata ครบมีทั้งสอง arrays=[]
missing_fields ใน A ไม่ใช่การรับรองว่าประเมินเงื่อนไข B/C ครบแล้ว

### JSON ที่รันได้จาก fixture ใหม่

| prefix ใน examples/gpu-storage-a/ | HTTP | Overall / ยอดเพิ่ม |
|---|---:|---|
| validate | 200 | unknown / 8000 |
| validate-unsupported | 200 | unknown / 6500; known USB |
| validate-cpu-conflict | 200 | failed / 5000 |
| validate-ram-conflict | 200 | failed / 5000; owned RAM ยังตรวจ type |
| validate-null-metadata | 200 | unknown / 700; quantity=3/owned=2 |
| validate-storage-limit | 400 | UNSUPPORTED_BUILD_SCOPE; USB 9 ลูกแม้ owned ทั้งหมด |
| validate-gpu-limit | 400 | UNSUPPORTED_BUILD_SCOPE; GPU 2 ใบแม้ owned ทั้งหมด |
| validate-not-found | 404 | NOT_FOUND ชนะ limit |

แต่ละ prefix มี `.request.json` และ `.response.json` แบบเต็มที่ tests เทียบกับ API
ตัวอย่าง B/C ใน PR #3 ไม่ใช่ผล A: ต่อให้ Board B มีแต่ SATA และเลือก NVMe
A ก็ยังคืน motherboard unknown ไม่คัดลอก hardware-failed response ของ B มาใช้

### Errors / CORS

error envelope คง `{"error":{"code":"...","message":"..."}}`
400 VALIDATION_ERROR: body/UUID/counts/duplicates/overflow/malformed query/duplicate mode
400 UNSUPPORTED_VALIDATION_MODE: mode ว่าง/ไม่รู้จัก (ส่งใน query เท่านั้น)
400 UNSUPPORTED_BUILD_SCOPE, 404 NOT_FOUND, 405 METHOD_NOT_ALLOWED, 413 PAYLOAD_TOO_LARGE,
415 UNSUPPORTED_MEDIA_TYPE, 500 INTERNAL_ERROR ตามสัญญา; conflict/unknown ตอบ 200

```bash
curl -i -X OPTIONS \
  -H 'Origin: http://localhost:3000' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type' \
  'http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1'
```

ควรได้ 204 และ Allow-Origin ตรงกัน; ทดลอง Origin http://127.0.0.1:3000 ได้เช่นกัน

## Tests / ขอบเขตที่ยังไม่เปิด

```bash
go test ./...
go vet ./...
go test -race ./internal/...
```

ใช้ httptest/snapshot/fixture ไม่มี Supabase/secrets; lifecycle tests เปิด loopback ชั่วคราว
ครอบคลุม typed null vs [], provenance, immutable nested copies, list/detail/shared prices,
limits/MaxInt64/owned/unsupported/input tampering/404 precedence, CPU/RAM failures,
legacy 11500, opt-in echo/status/coverage และ runnable JSON ทุกชุด

ยังไม่ทำ B/C hardware checks, allocation/bandwidth/BIOS/PSU/Case/boot/ความครบหรือ publish
ไม่มี typed PSU/Case contract ใหม่ ไม่มีการเดาสเปกสินค้าจริงหรือสูตร hardware
migrations 001/002 เดิมเป็น CPU/MB/RAM; metadata ใหม่อยู่ใน typed in-memory snapshot เท่านั้น
ไม่เพิ่ม/execute SQL migration หรือเก็บข้อมูลนี้ใน Supabase โดยปริยาย
database schema/persistence/RLS และ manufacturer-reviewed real SKU ingestion เป็นงานที่ต้องอนุมัติแยก
