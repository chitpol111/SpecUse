# SpecUse: Build Validation API สำหรับ Frontend

ใช้สัญญา response ที่ยืนยันกับ Fluke แล้ว Backend เป็นผู้คำนวณราคาและ compatibility
จาก catalog repository เดียวกับ GET products ทั้งหมดเป็น local fixture ยังไม่ใช้ Supabase

ล่าสุด [B1 SATA-first](STORAGE_B1.md) ตรวจ direct SATA allocation ใน opt-in เดิมได้เมื่อข้อมูล/หลักฐานครบ
ใช้ fixture และ runnable examples แยก ส่วน legacy response ในเอกสารนี้ยังคงเดิม

GPU/Storage ช่วง A ตาม contract PR #3 commit 3581447 เปิดใช้แล้วผ่าน opt-in query
อ่าน [GPU_STORAGE_A.md](GPU_STORAGE_A.md) สำหรับ synthetic fixture แยก, metadata, limits,
status/coverage และ runnable JSON/curl ส่วน POST response ด้านล่างเป็น **default legacy** ตามเดิม

## รัน Backend

ติดตั้ง Go 1.22 ขึ้นไปตาม `go.mod` และตรวจว่า `go version` ใช้งานได้ใน terminal
Backend ไม่มี third-party Go dependencies ไม่ต้องติดตั้ง Node/npm, PostgreSQL, Docker
หรือเตรียมบัญชี/key ของ Supabase เพื่อทดลอง local fixture
คำสั่ง curl ด้านล่างต้องมี curl; บน Windows ใช้ `curl.exe` เพื่อไม่สับสนกับ PowerShell alias

รับโฟลเดอร์ `backend/` จาก branch `feat/backend-api` ให้ครบ รวม `internal/`, `testdata/`,
`examples/` และ `go.mod` แนะนำ clone แยกสำหรับ Backend เพื่อไม่เปลี่ยน branch งาน Frontend
ดูขั้นตอนรับโค้ดและรายการไฟล์ใน [HANDOFF.md](HANDOFF.md)

บน macOS/Linux หรือ Git Bash จาก root ของ repository/ชุด ZIP ที่แตกแล้ว:

```sh
cd backend
APP_ENV=local \
LOCAL_CATALOG_FILE=./testdata/catalog.local.json \
PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

บน Windows PowerShell จาก root ของ repository/ชุด ZIP ที่แตกแล้ว:

```powershell
cd backend
$env:APP_ENV = 'local'
$env:LOCAL_CATALOG_FILE = './testdata/catalog.local.json'
$env:PORT = '8080'
$env:CORS_ALLOWED_ORIGINS = 'http://localhost:3000,http://127.0.0.1:3000'
go run ./cmd/api
```

เก็บ terminal นี้ไว้ขณะทดสอบ และเปิด terminal อีกหน้าสำหรับเรียก API กด Ctrl+C เพื่อหยุด
ค่า environment ใน PowerShell จะคงอยู่ใน terminal นั้น ส่วนแบบ Bash ด้านบนตั้งเฉพาะ process ที่รัน

- Base URL: `http://127.0.0.1:8080/api/v1`
- Port เริ่มต้น: `8080` เปลี่ยนได้ผ่าน `PORT`
- Frontend ที่ยืนยันใช้ port 3000: ตั้ง CORS ทั้ง `http://localhost:3000` และ `http://127.0.0.1:3000`
- local mode bind เฉพาะ `127.0.0.1`; ใช้ Base URL ข้างต้นเพื่อไม่ให้ localhost resolve เป็น IPv6
- ไม่มีการโหลด `.env` อัตโนมัติ ใช้ environment ของ process หรือค่าจากคำสั่งข้างต้น
- ต้องระบุทั้ง APP_ENV และ fixture path หากไม่ระบุ path catalog จะว่าง
  หากระบุ path ใน environment ที่ไม่ใช่ local จะปฏิเสธการเริ่ม server
- fixture ใช้ `specs` และ `ram_model_id` ที่เชื่อถือได้จากไฟล์บน Backend เท่านั้น
  แก้ custom fixture ของ CPU/motherboard/RAM ให้มี metadata ครบก่อนใช้ loader รุ่นนี้
- โปรแกรมไม่ embed fixture และไม่โหลดข้อมูลสมมติใน production โดยอัตโนมัติ
- endpoint เป็น guest ไม่ใช้ token, cookies หรือ `credentials: include`

### เมื่อ Frontend ใช้ port อื่น

ใช้ origin ตาม URL จริงบน browser: protocol + hostname + port โดยไม่มี path หรือ slash ท้าย
`localhost` และ `127.0.0.1` เป็นคนละ origin แม้ใช้ port เดียวกัน
ตัวอย่าง React ใช้ port 3001 ให้เปลี่ยนค่าขณะเริ่ม Backend เป็น:

```text
CORS_ALLOWED_ORIGINS=http://localhost:3001,http://127.0.0.1:3001
```

ให้แก้ค่าในคำสั่ง Bash หรือ `$env:CORS_ALLOWED_ORIGINS` ของ PowerShell แล้วเริ่ม Backend ใหม่
การเปลี่ยน port ของ React ไม่เปลี่ยน Base URL ของ API หาก Backend ยังใช้ port 8080
หาก port 8080 ไม่ว่าง ให้ตั้ง `PORT=8081` และเปลี่ยน API Base URL เป็น
`http://127.0.0.1:8081/api/v1` ให้ตรงกัน

## Endpoints ที่ใช้ได้ในชุดนี้

| Method/path ต่อจาก Base URL | Request | Response สำเร็จ |
|---|---|---|
| `GET /products` | query `category`, `search`, `page`, `page_size` ไม่ส่ง body | `data.{items,page,page_size,total}` |
| `GET /products/{id}` | product UUID ใน path ไม่ส่ง body | `data.product` |
| `POST /builds/validate` | JSON `items` และ `ram_items` ตามรายละเอียดด้านล่าง | `data.{compatible,issues,items,ram_selections,total_price_thb}` |
| `POST /builds/validate?validation_mode=gpu_storage_v1` | body เดิม | fields เดิม + `data.compatibility` ตาม [B1](STORAGE_B1.md) |

Catalog กรองเฉพาะ `is_active=true`; ชื่อ field ยังคงเป็น `is_active` ไม่ใช่ `active`
page เริ่มที่ 1 และ page_size เริ่มต้น 20 ต้องเป็นจำนวนเต็มบวก; หากต้องการค่าเริ่มต้น
ให้ละ query นั้น ไม่ส่งค่าว่าง เช่น `page=` category เทียบตรงตัว ส่วน search ค้นบางส่วน
ของชื่อแบบไม่แยกตัวพิมพ์เล็ก/ใหญ่ เรียง UUID ก่อนแบ่งหน้า

Catalog เพิ่ม fields แบบ additive ทั้งรายการและรายละเอียด โดยคง `id`, `name`,
`category`, `brand`, `price_thb`, `is_active` และ pagination/envelope เดิม:

| Category | Fields ที่เพิ่ม | ความหมาย |
|---|---|---|
| `cpu` | `specs.cpu_socket` | socket ของ CPU |
| `motherboard` | `specs.cpu_socket`, `specs.ram_type`, `specs.connectivity_profiles`, `specs.sources`, `specs.evidence` | socket/RAM type เดิม พร้อม nullable metadata ช่วง A |
| `ram` | `ram_model_id`, `specs.ram_type`, `specs.modules_per_package` | UUID ของ RAM model, RAM type และจำนวนโมดูลต่อแพ็ก 1 หรือ 2 |
| `gpu`, `storage` | `specs.display`, `specs.host`, `specs.power`, `specs.dimensions`, `specs.sources`, `specs.evidence` | typed nullable ตาม [ช่วง A](GPU_STORAGE_A.md); ไม่รับรองผ่าน hardware checks |
| หมวดอื่น | ไม่มี fields เพิ่ม | PSU/Case/cooler ยังไม่มี typed specs |

`ram_model_id` อยู่ระดับเดียวกับ `id` ไม่ได้อยู่ใน `specs` และไม่ใช่ product UUID
SKU แพ็กเดี่ยว/คู่ของ RAM model เดียวกันใช้ ram_model_id เดียวกัน แต่มี product id/ราคา
และ modules_per_package ของตัวเอง ตัวอย่างแพ็กเดี่ยว/คู่ด้านล่างอ่านจาก catalog เดียวกับ validate

CPU/RAM specs ส่ง keys เดิม ไม่เติมค่าว่าง/null หรือ modules_per_package=0
GPU/Storage และ metadata ใหม่ของ motherboard ใช้ null เมื่อยังไม่รู้, [] เมื่อ reviewed empty
สำหรับหมวดอื่นที่ยังไม่มี typed specs จะละ `specs` และ `ram_model_id` ทั้งคู่
category ยังคงเป็นตัวพิมพ์เล็กและสถานะยังชื่อ `is_active`

ตัวอย่าง CPU จาก local fixture:

```json
{"data":{"product":{"id":"abcdefab-0000-0000-0000-000000000001","name":"[LOCAL TEST] CPU A","category":"cpu","brand":"Example","price_thb":25000,"is_active":true,"specs":{"cpu_socket":"AM5"}}}}
```

ตัวอย่าง RAM แพ็กคู่จาก local fixture:

```json
{"data":{"product":{"id":"abcdefab-0000-0000-0000-000000000006","name":"[LOCAL TEST] DDR5 pair","category":"ram","brand":"Example","price_thb":4000,"is_active":true,"specs":{"ram_type":"DDR5","modules_per_package":2},"ram_model_id":"10000000-0000-0000-0000-000000000001"}}}
```

ตัวอย่าง JSON เต็มที่ตรวจเทียบกับ API ใน tests:

- [รายการสินค้าทั้งหมด](examples/products.response.json)
- [CPU](examples/product-cpu.response.json)
- [Motherboard](examples/product-motherboard.response.json)
- [RAM แพ็กเดี่ยว](examples/product-ram-single.response.json) และ [RAM แพ็กคู่](examples/product-ram-pair.response.json)

### ใช้ข้อมูล GET เพื่อสร้าง request validate

1. อ่าน `data.items` จาก GET products และแสดงสินค้าโดยใช้ 6 fields เดิม
2. CPU/motherboard ใช้ `product.id` ใน `items[].product_id` ตามเดิม
3. จัดกลุ่ม RAM SKUs ด้วย `ram_model_id` จาก response โดยตรง ไม่ทำ mapping UUID เอง
4. เมื่อเลือกรุ่น RAM ส่ง model UUID นั้นใน `ram_items[].ram_model_id` พร้อมจำนวนโมดูล
   ที่ต้องการและมีอยู่แล้ว ไม่ส่ง product id ของแพ็กเป็น RAM model ID
5. ไม่ต้องเลือกแพ็ก RAM เอง Backend เลือก active SKU ที่เหมาะสมแล้วส่งกลับใน `ram_selections[].packages`

Frontend ใช้ specs ช่วยแสดง/กรองตัวเลือกได้ แต่ผล compatibility และราคาสุดท้ายต้องใช้จาก POST validate
ห้ามส่ง product ทั้ง object หรือ specs/ราคาใน request เพราะ endpoint รับเฉพาะ IDs/counts
ไม่มี route เพิ่มสำหรับ metadata: ทั้ง GET list และ GET detail มีข้อมูลที่จำเป็นชุดเดียวกัน

Catalog ตอบ 200 เมื่อสำเร็จ, 400 เมื่อ UUID/query ผิด, 404 เมื่อไม่พบหรือสินค้า inactive,
405 เมื่อ method ไม่ใช่ GET และ 500 เมื่อเกิด internal error โดยใช้ error envelope เดียวกัน
GET ที่รายการว่างหรือ page เกินจริงยังตอบ 200 พร้อม `items:[]`
เส้นทางต้องไม่มี slash ท้าย เช่น `/products` และ `/builds/validate`

## `POST /api/v1/builds/validate`

ส่ง `Content-Type: application/json` โดย body เป็น object เดียว ขนาดไม่เกิน 1 MiB
ต้องมี `items` และ `ram_items` เป็น arrays; ส่ง `[]` สำหรับรายการที่ยังไม่ได้เลือก

| Request field | กฎ |
|---|---|
| `items[].product_id` | UUID ของสินค้า non-RAM ที่ active |
| `items[].quantity` | จำนวนเต็ม >= 1 |
| `items[].owned_quantity` | จำนวนเต็มตั้งแต่ 0 ถึง quantity |
| `ram_items[].ram_model_id` | UUID ของ RAM model ไม่ใช่ product UUID |
| `ram_items[].required_modules` | จำนวนเต็ม >= 1 นับเป็นโมดูล ไม่ใช่จำนวนแพ็ก |
| `ram_items[].owned_modules` | จำนวนเต็มตั้งแต่ 0 ถึง required_modules |

ทุก field ในแต่ละ entry ต้องมีค่า ไม่รับ null/จำนวนทศนิยม ห้ามส่ง product/model เดียวกันซ้ำ
สินค้า RAM ต้องอยู่ใน `ram_items` เพื่อให้ Backend เลือก SKU/จำนวนแพ็กให้
ไม่รับ fields เพิ่มที่ไม่อยู่ในสัญญา เช่น `price_thb`, `total_price_thb`, `compatible`, `specs` หรือ role
หากส่งมาจะตอบ 400 แทนการใช้ข้อมูลนั้นคำนวณ

ตัวอย่างพร้อมใช้: [request](examples/validate.request.json) และ [response](examples/validate.response.json)

```json
{
  "items": [
    {"product_id":"abcdefab-0000-0000-0000-000000000001","quantity":1,"owned_quantity":1},
    {"product_id":"abcdefab-0000-0000-0000-000000000004","quantity":1,"owned_quantity":0}
  ],
  "ram_items": [
    {"ram_model_id":"10000000-0000-0000-0000-000000000001","required_modules":3,"owned_modules":0}
  ]
}
```

ตัวอย่างนี้มี CPU อยู่แล้ว จึงคิดเฉพาะ motherboard 5,000 บาท และ RAM 6,500 บาท
RAM ซื้อแพ็กเดี่ยว 1 แพ็กกับแพ็กคู่ 1 แพ็ก รวม 3 โมดูลพอดี ยอดรวม 11,500 บาท

## Response: HTTP 200

รูปแบบหลัก:

```json
{
  "data": {
    "compatible": true,
    "issues": [],
    "items": [
      {"product_id":"abcdefab-0000-0000-0000-000000000001","quantity":1,"owned_quantity":1,"purchase_quantity":0,"unit_price_thb":25000,"purchase_price_thb":0},
      {"product_id":"abcdefab-0000-0000-0000-000000000004","quantity":1,"owned_quantity":0,"purchase_quantity":1,"unit_price_thb":5000,"purchase_price_thb":5000}
    ],
    "ram_selections": [
      {
        "ram_model_id":"10000000-0000-0000-0000-000000000001",
        "required_modules":3,
        "owned_modules":0,
        "purchase_modules":3,
        "supplied_modules":3,
        "excess_modules":0,
        "packages":[
          {"product_id":"abcdefab-0000-0000-0000-000000000005","quantity":1,"modules_per_package":1,"unit_price_thb":2500},
          {"product_id":"abcdefab-0000-0000-0000-000000000006","quantity":1,"modules_per_package":2,"unit_price_thb":4000}
        ],
        "total_price_thb":6500
      }
    ],
    "total_price_thb":11500
  }
}
```

- `compatible` ตรวจเฉพาะ CPU socket กับ motherboard และ RAM type กับ motherboard
- `issues` เป็น array ของ `{code,message}`; ไม่มีปัญหาตอบ `[]`
- `items[].purchase_quantity` คือ quantity - owned_quantity
- `items[].unit_price_thb` คือราคาสินค้าปัจจุบัน; `purchase_price_thb` คือราคาที่ต้องซื้อเพิ่ม
- `purchase_modules` คือ required_modules - owned_modules
- `supplied_modules` คือจำนวนโมดูลรวมจากแพ็กที่ Backend เลือกซื้อใหม่ ไม่รวม owned_modules
- `excess_modules` คือ supplied_modules - purchase_modules
- `packages[].quantity` เป็นจำนวนแพ็ก ส่วน `modules_per_package` เป็นจำนวนโมดูลต่อแพ็ก
- `ram_selections[].total_price_thb` รวมราคาแพ็กที่ต้องซื้อสำหรับ model นั้น
- `data.total_price_thb` รวม purchase_price_thb ของ non-RAM และยอดซื้อแพ็ก RAM เท่านั้น
- ทุกค่าเงินเป็นจำนวนเต็มเงินบาท Frontend จัดรูปแบบ เช่น `11,500 ฿` เอง
- owned RAM ยังมีผลต่อ compatibility แม้ซื้อเพิ่ม 0 โมดูล กรณีนี้ packages เป็น `[]` และราคา RAM เป็น 0

### อุปกรณ์ไม่เข้ากัน

ยังตอบ HTTP 200 เพราะประเมินสำเร็จ แต่ `compatible:false` และมี issues:

```json
[
  {"code":"CPU_SOCKET_MISMATCH","message":"CPU และเมนบอร์ดใช้ซ็อกเก็ตต่างกัน"},
  {"code":"RAM_TYPE_MISMATCH","message":"RAM และเมนบอร์ดใช้ RAM type ต่างกัน"}
]
```

ดู [request](examples/validate-incompatible.request.json) และ [response แบบเต็ม](examples/validate-incompatible.response.json)
ที่ทดสอบกับ fixture แล้ว รายการนี้มียอดซื้อเพิ่ม 9,500 บาท

### RAM selection

เลือกจำนวนโมดูลพอดีก่อน ถ้าพอดีไม่ได้เลือกส่วนเกินน้อยที่สุด แล้วเลือกยอดราคาต่ำที่สุด
ราคาเท่ากันเลือกจำนวนแพ็กน้อยที่สุด; SKU ขนาดเดียวกันราคาเท่ากันเลือก UUID ที่เรียงมาก่อน
เลือกเฉพาะ active SKU ของ model เดียวกัน และคืน packages เรียง UUID
ยังไม่มี stock limits จึงเลือก SKU เดิมซ้ำหลายแพ็กได้

ดูตัวอย่าง [request กรณีมีแต่แพ็กคู่](examples/validate-excess.request.json) และ
[response](examples/validate-excess.response.json): ต้องซื้อ 3 โมดูล แต่มีเฉพาะแพ็กคู่ active
จึงเลือก 2 แพ็ก ได้ 4 โมดูล ส่วนเกิน 1 โมดูล ไม่เลือกแพ็กเดี่ยวที่ inactive แม้ราคาถูกกว่า

### Partial draft

ยังไม่กำหนดหมวดบังคับหรือเกณฑ์ความครบ `{ "items": [], "ram_items": [] }` ตอบ:

```json
{"data":{"compatible":true,"issues":[],"items":[],"ram_selections":[],"total_price_thb":0}}
```

`compatible:true` ไม่ยืนยันว่า build ครบ และไม่ใช่สิทธิ์เผยแพร่ public
endpoint นี้ไม่บันทึก build; ยังไม่มี save/member/admin/publication APIs

## Errors

```json
{"error":{"code":"VALIDATION_ERROR","message":"ข้อมูลรายการอุปกรณ์หรือจำนวนไม่ถูกต้อง"}}
```

| HTTP | Code | กรณี |
|---|---|---|
| 400 | `VALIDATION_ERROR` | JSON/input ผิด, arrays/counts หายหรือ null, UUID ผิด, จำนวนผิด/overflow, duplicate entries, RAM SKU อยู่ใน items, fields ที่ไม่รองรับ |
| 404 | `NOT_FOUND` | ไม่พบ active product, RAM model ไม่พบ หรือไม่มี active SKU สำหรับโมดูลที่ต้องซื้อ |
| 405 | `METHOD_NOT_ALLOWED` | endpoint validate รับ POST เท่านั้น พร้อม `Allow: POST` |
| 413 | `PAYLOAD_TOO_LARGE` | body เกิน 1 MiB |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | ไม่ได้ส่ง Content-Type: application/json |
| 500 | `INTERNAL_ERROR` | repository/server error โดยไม่ส่งรายละเอียดภายใน |

สินค้าที่ inactive ไม่ถูกเปิดเผยผ่าน GET และใช้ใน non-RAM validation ไม่ได้
การจัดการ build เก่าที่มีสินค้าถูก deactivate ยังเป็น open decision สำหรับ save APIs

## curl จากโฟลเดอร์ backend

```sh
# Catalog คง 6 fields เดิมและเพิ่ม specs/ram_model_id ตามประเภทสินค้า
curl -i 'http://127.0.0.1:8080/api/v1/products'

# รายละเอียดสินค้า
curl -i 'http://127.0.0.1:8080/api/v1/products/abcdefab-0000-0000-0000-000000000001'

# RAM SKUs: อ่าน ram_model_id และ modules_per_package โดยไม่ hardcode mapping
curl -i 'http://127.0.0.1:8080/api/v1/products?category=ram'
curl -i 'http://127.0.0.1:8080/api/v1/products/abcdefab-0000-0000-0000-000000000006'

# สำเร็จ: compatible=true, total_price_thb=11500
curl -i -X POST 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://localhost:3000' \
  --data-binary @examples/validate.request.json

# CPU และ RAM ไม่เข้ากัน: HTTP 200, compatible=false
curl -i -X POST 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Content-Type: application/json' \
  --data-binary @examples/validate-incompatible.request.json

# โมดูลส่วนเกินน้อยที่สุด
curl -i -X POST 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Content-Type: application/json' \
  --data-binary @examples/validate-excess.request.json

# ข้อมูลผิด: HTTP 400
curl -i -X POST 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Content-Type: application/json' \
  --data '{"items":[{"product_id":"bad","quantity":1,"owned_quantity":0}],"ram_items":[]}'

# POST preflight ของ React: HTTP 204
curl -i -X OPTIONS 'http://127.0.0.1:8080/api/v1/builds/validate' \
  -H 'Origin: http://127.0.0.1:3000' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type'
```

Preflight อนุญาต `GET, POST` และ `Authorization, Content-Type` สำหรับ origins ที่ตั้งไว้
คำสั่ง curl ไม่บังคับ CORS เหมือน browser ให้ดู `Access-Control-Allow-Origin` จาก `-i`
ประกอบด้วย หาก Origin ตรงค่าที่ตั้งไว้ Backend จะส่ง header นี้กลับมา
Frontend ใช้ fetch POST พร้อม JSON.stringify(request) แล้วอ่าน `response.data` จาก parsed body
เมื่อ HTTP ไม่สำเร็จให้อ่าน `error.code`/`error.message` แยกจาก compatibility issues

บน Windows PowerShell เปิด terminal อีกหน้า เข้า `backend/` แล้วใช้คำสั่งบรรทัดเดียว:

```powershell
curl.exe -i -X POST 'http://127.0.0.1:8080/api/v1/builds/validate' -H 'Content-Type: application/json' -H 'Origin: http://localhost:3000' --data-binary '@examples/validate.request.json'
```

อย่าใช้ backslash ต่อบรรทัดของตัวอย่าง Bash ใน PowerShell
หาก browser อ่าน response ไม่ได้ แต่ curl ได้ ให้ตรวจ Origin ที่ browser ใช้และ CORS setting
หากรายการว่าง ให้ตรวจว่าเริ่ม server ด้วย APP_ENV/LOCAL_CATALOG_FILE ถูกต้องและใช้ fixture ชุดเดียวกัน

## Tests และขอบเขต

```sh
go test ./...
go vet ./...
```

Tests ครอบคลุม handler/service/repository และตรวจ JSON examples ของ Catalog กับ validate ทั้งสามชุดกับ
API ที่สร้างจาก local fixture จริงด้วย `httptest` ไม่ต้องใช้ Supabase หรือ secrets
Lifecycle tests เดิมเปิดเฉพาะพอร์ต loopback ชั่วคราว ไม่มีการเชื่อมบริการภายนอก
GET Catalog และ POST validate อ่าน snapshot เดียวกัน รวมถึง IDs, specs, ราคา และ active status
มี tests สร้าง request validate จากข้อมูล GET โดยตรง ตรวจ CPU/RAM mismatch และแพ็ก 1/2 โมดูล
รวมถึงป้องกันการแก้ specs ของ response แล้วกระทบ snapshot และไม่ส่ง specs ของหมวดที่ยังไม่ตกลง
POST success example เดิมยังได้ compatible=true และ total_price_thb=11500
โหมด opt-in ของ payload เดียวกันยังยอด 11500 แต่ status=unknown/compatible=false เพราะ PSU/Case ยังไม่ตรวจ
tests เพิ่มสำหรับช่วง A/ขอบเขต/JSON ใหม่อธิบายใน [GPU_STORAGE_A.md](GPU_STORAGE_A.md)
