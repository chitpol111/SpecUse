# SpecUse Backend — Catalog, Build Validation, Supabase และสมาชิก

Go REST API ด้วย `net/http` และ JSON แยก handler, service และ repository
มี public catalog สอง endpoints ซึ่งคง 6 fields เดิมและเพิ่ม compatibility metadata ตามประเภทสินค้า
รวมถึง `POST /api/v1/builds/validate` จากข้อมูล repository ชุดเดียวกัน
ไม่มี third-party Go dependencies; เพิ่ม Supabase Auth/Data API adapter แล้ว แต่ยังไม่ได้ทดสอบ hosted project หรือ deploy จริง

เพิ่ม private draft ใน branch แยก ดู [PRIVATE_BUILDS_API.md](PRIVATE_BUILDS_API.md)
ยังไม่ apply migration 007 กับ specuse-dev และยังไม่ deploy งานนี้

เริ่มงานสมาชิกด้วย [AUTH_API.md](AUTH_API.md), ตั้งค่า DB ด้วย [SUPABASE_SETUP.md](SUPABASE_SETUP.md),
สิทธิ์สินค้าใน [ADMIN_API.md](ADMIN_API.md) และเตรียม deploy ด้วย [RENDER.md](RENDER.md)
Register ได้ member/session พร้อมใช้ทันที ไม่ต้อง OTP; /auth/confirm เดิมตอบ 410
React เรียก Go เท่านั้น; local fixture ด้านล่างยังใช้ได้โดยไม่ต้องมีบัญชีหรือ keys

ล่าสุดเพิ่ม [B1 SATA-first: วิธีรันและ JSON/curl](STORAGE_B1.md) สำหรับ synthetic fixture แยก
ใช้ opt-in/schema เดิม; legacy response และราคา 11500 คงเดิม ยังไม่ตรวจ GPU/M.2/PSU/Case

เพิ่ม [GPU/Storage ช่วง A: วิธีรัน, contract และ JSON/curl](GPU_STORAGE_A.md)
ตาม PR #3 commit 3581447 แล้ว โดย opt-in เท่านั้น; default validate และตัวอย่างเดิมยอด 11500 คงเดิม

อ่าน [API Contract สำหรับ Fluke พร้อม JSON/curl](BUILD_API.md) และ
[Phase 2 schema/repository](PHASE2.md) สำหรับ compatibility และ RAM selection
ดู [ชุดส่งต่อให้ฟลุค](HANDOFF.md) สำหรับ setup, รายการไฟล์ที่จะ commit และร่างข้อความส่งต่อ

## โครงสร้าง

```text
backend/
  cmd/api/                        HTTP server, local fixture loader, lifecycle และ API wiring tests
  internal/catalog/               Product, repository interface, service และ tests
  internal/handler/               HTTP/JSON, CORS และ tests
  internal/repository/            repository ในหน่วยความจำและ tests
  internal/account/               Auth/profile service และ ownership checks
  internal/supabase/              Auth HTTP client และ database RPC repository
  internal/management/            Admin product service
  migrations/001–007*.sql         catalog, profiles, restricted RPC และ health
  integration/                    disposable PostgreSQL permissions/CRUD tests
  render.yaml                     Render Blueprint (ยังไม่ได้สร้างบริการ)
  testdata/catalog.local.json     ข้อมูลสมมติสำหรับ local แบบ opt-in
  examples/                       Request/response JSON ที่ทดสอบกับ fixture แล้ว
  .env.example                    ตัวอย่าง configuration ที่ไม่มี secrets
```

## รันในเครื่อง

ใช้ Go 1.22 ขึ้นไป จาก root ของ repository เปิดข้อมูลสมมติสำหรับ local
โดยระบุทั้ง `APP_ENV=local` และ `LOCAL_CATALOG_FILE` อย่างชัดเจน:

```sh
cd backend
APP_ENV=local \
LOCAL_CATALOG_FILE=./testdata/catalog.local.json \
PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

Environment variables:

| ตัวแปร | ค่าเริ่มต้น | ความหมาย |
|---|---|---|
| `PORT` | `8080` | HTTP port จำนวนเต็ม 1–65535 |
| `CORS_ALLOWED_ORIGINS` | ว่าง | origin ที่อนุญาต คั่นด้วย comma เช่น `http://localhost:3000,http://localhost:3001` |
| `APP_ENV` | ว่าง | ต้องเป็น `local` แบบตรงตัวจึงเปิดใช้ fixture ได้; โหมดนี้ bind เฉพาะ `127.0.0.1` |
| `CATALOG_BACKEND` | `memory` | เลือก `supabase` อย่างชัดเจนเมื่อใช้ DB/Auth; production บังคับ Supabase |
| `LOCAL_CATALOG_FILE` | ว่าง | path ของ JSON fixture อ้างอิงจาก working directory; ต้องใช้คู่กับ `APP_ENV=local` |

ระบุ origin แบบตรงตัว ไม่มี path หรือ slash ท้าย origin ไม่รองรับ wildcard `*`
หากไม่ตั้งค่า จะไม่มี CORS headers อนุญาต browser จาก origin อื่น
โปรแกรมไม่อ่านไฟล์ `.env` อัตโนมัติ ต้อง export variables ผ่าน shell หรือ deployment configuration
ตัวแปร Supabase ทั้งหมดดู [SUPABASE_SETUP.md](SUPABASE_SETUP.md); ห้ามตั้งร่วมกับ fixture

### การแยกข้อมูล local จาก production

- ค่าเริ่มต้นเมื่อไม่ตั้ง `LOCAL_CATALOG_FILE` คือ catalog ว่าง แม้ตั้ง `APP_ENV=local`
- ตั้ง path ของ fixture โดยไม่ใช้ `APP_ENV=local` จะหยุดพร้อม error ก่อนอ่านไฟล์
- โหมด local รับการเชื่อมต่อที่ `127.0.0.1` เท่านั้น ไม่มีการ embed fixture ลง executable
- JSON fixture ผิดรูปแบบ, มี fields ที่ไม่รู้จัก, ราคาไม่ใช่จำนวนเต็ม หรือข้อมูลสินค้าไม่ถูกต้อง จะหยุดพร้อม error
- สำหรับ production ให้เว้น `LOCAL_CATALOG_FILE`, ใช้ `APP_ENV=production` และ `CATALOG_BACKEND=supabase`
  พร้อมตั้งค่าตามคู่มือ; โปรแกรมปฏิเสธ production memory และไม่ fallback เป็นข้อมูลสมมติเมื่อ DB ล่ม
- fixture มีสินค้า active 7 รายการและ inactive 2 รายการ ทุกชื่อขึ้นต้นด้วย `[LOCAL TEST]`
  สินค้าและราคาเป็นข้อมูลสมมติ ค่า `cpu`, `motherboard`, `ram` เป็นเพียงตัวอย่าง ไม่ใช่ enum ที่เพิ่มในสัญญา

หากต้องการทดลองค่าเริ่มต้นแบบไม่มีข้อมูล ให้หยุด server เดิมก่อน แล้วจาก `backend/` รัน:

```sh
env -u LOCAL_CATALOG_FILE -u SUPABASE_URL APP_ENV=local CATALOG_BACKEND=memory PORT=8080 go run ./cmd/api
```

Supabase mode อ่าน snapshot ผ่าน Data API RPC เดียวร่วมกันระหว่าง Catalog/Validate; Admin เขียนลง DB ผ่าน restricted RPC

### Graceful shutdown

กด Ctrl+C หรือส่ง SIGTERM ให้ process ของ API โปรแกรมจะหยุดรับ connection ใหม่
และรอคำขอที่กำลังทำงานให้เสร็จก่อนออกจาก `main` โดยรอสูงสุด 15 วินาที
หากเกินเวลาจะปิด connections ที่เหลือและรายงาน shutdown error

## ทดลองเชื่อม React และเรียก API

ตัวอย่าง CORS สำหรับ React ที่เปิดผ่าน `http://localhost:3000`:

```sh
export CORS_ALLOWED_ORIGINS=http://localhost:3000
```

ถ้าเปิด Frontend ผ่านหลาย origin ต้องระบุแยกกัน เช่น:

```sh
export CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001
```

ใช้ API base URL `http://127.0.0.1:8080/api/v1` สำหรับคำสั่ง local ข้างต้น
ไม่ต้องส่ง token สำหรับสอง endpoints นี้ และไม่ต้องใช้ cookies/`credentials: include`
ให้ Frontend อ่าน `data.items`/`data.product` และจัดรูปแบบราคาเอง

เปิด terminal อีกหน้าขณะรัน server พร้อม fixture แล้วทดลอง:

```sh
# รายการ: HTTP 200, total=7, items มีเฉพาะสินค้า active
curl -i 'http://127.0.0.1:8080/api/v1/products'

# filter และ pagination: HTTP 200, total=2, items 1 รายการ
curl -i 'http://127.0.0.1:8080/api/v1/products?category=cpu&search=CPU&page=1&page_size=1'

# รายละเอียด: HTTP 200, data.product.price_thb=25000
curl -i 'http://127.0.0.1:8080/api/v1/products/abcdefab-0000-0000-0000-000000000001'

# สินค้า inactive: HTTP 404, error.code=NOT_FOUND
curl -i 'http://127.0.0.1:8080/api/v1/products/abcdefab-0000-0000-0000-000000000002'

# UUID ผิดรูปแบบ: HTTP 400, error.code=VALIDATION_ERROR
curl -i 'http://127.0.0.1:8080/api/v1/products/not-a-uuid'

# ตรวจ CORS: ต้องเห็น Access-Control-Allow-Origin: http://localhost:3000
curl -i -H 'Origin: http://localhost:3000' 'http://127.0.0.1:8080/api/v1/products'

# preflight: HTTP 204 ไม่มี body พร้อม Allow-Methods: GET, POST, PATCH, DELETE
curl -i -X OPTIONS \
  -H 'Origin: http://localhost:3000' \
  -H 'Access-Control-Request-Method: GET' \
  -H 'Access-Control-Request-Headers: authorization,content-type' \
  'http://127.0.0.1:8080/api/v1/products'
```

ใช้เส้นทางโดยไม่มี slash ท้าย endpoint และละ `page`/`page_size` หากต้องการค่าเริ่มต้น
อย่าส่ง pagination เป็นค่าว่าง เพราะจะตอบ 400

## API

Catalog ทั้งสอง endpoints และ build validation เปิดให้ guest และไม่มีการรับราคา/role จาก client
ราคาคือจำนวนเต็มเงินบาท (`int64` ใน Go, `BIGINT` ใน PostgreSQL)
ไม่มีหน่วยเงินหรือ comma ในค่า JSON

### `GET /api/v1/products?category=&search=&page=&page_size=`

- `page` เริ่มที่ 1 และมีค่าเริ่มต้น 1
- `page_size` มีค่าเริ่มต้น 20 ต้องเป็นจำนวนเต็มบวก
- ไม่ส่ง `page`/`page_size` ใช้ค่าเริ่มต้น แต่ส่งค่าว่างถือว่าข้อมูลผิด
- ปฏิเสธ pagination ที่เกินขอบเขต integer/offset และ query parameters ที่ระบุซ้ำสำหรับ 4 fields นี้
- `category` ว่างหมายถึงทุกประเภท ค่าอื่นเทียบตรงตัวกับค่าที่เก็บใน catalog
- `search` ว่างหมายถึงไม่ค้นหา ระยะนี้ค้นบางส่วนของชื่อสินค้าแบบไม่แยกตัวพิมพ์เล็ก/ใหญ่
- ตัดช่องว่างต้นและท้าย `category`/`search` ก่อนค้น
- เรียง UUID จากน้อยไปมากเพื่อให้ pagination คงที่สำหรับข้อมูลชุดเดิม
- กรอง active/category/search ก่อนนับ `total` และแบ่งหน้า
- ไม่พบผลลัพธ์หรือหน้าเกินจำนวนจริงตอบ 200 และ `items: []` โดย `total` ยังเป็นจำนวนที่ตรง filter ทั้งหมด

ตัวอย่างเมื่อ catalog ว่าง:

```json
{"data":{"items":[],"page":1,"page_size":20,"total":0}}
```

ตัวอย่างข้อมูลสมมติสำหรับอธิบายรูปแบบ ชื่อสินค้าใน local fixture ขึ้นต้นด้วย `[LOCAL TEST]`:

```json
{
  "data": {
    "items": [
      {
        "id": "abcdefab-0000-0000-0000-000000000001",
        "name": "[LOCAL TEST] CPU A",
        "category": "cpu",
        "brand": "Example",
        "price_thb": 25000,
        "is_active": true,
        "specs": {"cpu_socket": "AM5"}
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 1
  }
}
```

### `GET /api/v1/products/{id}`

รับ UUID รูปแบบ `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` ไม่บังคับ version
รับตัวอักษร hex พิมพ์ใหญ่/เล็กและ normalize เป็นพิมพ์เล็กก่อนค้น

ตัวอย่างจาก test fixture:

```json
{
  "data": {
    "product": {
      "id": "abcdefab-0000-0000-0000-000000000001",
      "name": "[LOCAL TEST] CPU A",
      "category": "cpu",
      "brand": "Example",
      "price_thb": 25000,
      "is_active": true,
      "specs": {"cpu_socket": "AM5"}
    }
  }
}
```

UUID ผิดรูปแบบตอบ 400 สินค้าไม่มีอยู่หรือ inactive ตอบ 404 รูปแบบเดียวกัน
public API ไม่คืน inactive; Admin จัดการได้ผ่าน protected endpoints ใน [ADMIN_API.md](ADMIN_API.md)

### Errors

```json
{"error":{"code":"NOT_FOUND","message":"ไม่พบสินค้า"}}
```

| HTTP | Code | กรณี |
|---|---|---|
| 400 | `VALIDATION_ERROR` | UUID/query/pagination ไม่ถูกต้อง |
| 404 | `NOT_FOUND` | ไม่พบสินค้าที่ active หรือไม่พบ route |
| 405 | `METHOD_NOT_ALLOWED` | method ที่ไม่ใช่ GET บน catalog route พร้อม `Allow: GET` |
| 500 | `INTERNAL_ERROR` | repository/server error โดยไม่ส่งรายละเอียดภายในให้ client |

การตอบ preflight CORS เป็น 204 ไม่มี body เมื่อ origin อยู่ในรายการอนุญาต
routes ที่ไม่ได้ระบุ เช่น trailing slash ไม่ redirect และตอบ JSON 404

## Schema และ migration

`migrations/001_products.up.sql` สร้างเฉพาะ `public.products`:

| Column | Type/constraint |
|---|---|
| `id` | UUID primary key, default `gen_random_uuid()` |
| `name` | text, not null และไม่ว่าง |
| `category` | text, not null และไม่ว่าง ยังไม่กำหนด enum |
| `brand` | text, not null และไม่ว่าง |
| `price_thb` | bigint, not null, >= 0 |
| `is_active` | boolean, not null, default true |

มี partial index `(category, id)` สำหรับ active products และเปิด RLS โดยไม่เพิ่ม
policies ให้ anon/authenticated ในระยะนี้ การเปิด RLS ไม่มีผลแทนการกรอง active ใน Go
โดยเฉพาะเมื่อ backend ใช้ database role ที่ bypass RLS

migrations 001–007 ทดสอบด้วย disposable PostgreSQL และ auth schema/roles จำลอง
ยังไม่ทดสอบ hosted Supabase; ไม่มี migration อัตโนมัติตอนเปิด server
ตั้ง schema ตาม [SUPABASE_SETUP.md](SUPABASE_SETUP.md); ตัวอย่างคำสั่งรัน migration แรกกับฐานทดสอบ:
ตั้ง `DATABASE_URL` เป็น connection string ของฐานทดสอบผ่าน environment แล้วใช้:

```sh
cd backend
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 --single-transaction -f migrations/001_products.up.sql
```

คำสั่งนี้ใช้กับฐานทดสอบที่ยังไม่มีตาราง `products` และมี `gen_random_uuid()`
`DATABASE_URL` ใช้โดย `psql` เท่านั้น; API ใช้ Supabase URL/keys สำหรับ Auth และ Data API RPC
ห้ามใส่ connection strings, access tokens หรือ service-role keys จริงใน repository

## Tests

จากโฟลเดอร์ `backend/`:

```sh
go test ./...
go vet ./...
go test -race ./internal/...
# ต้องมี Docker; จำลอง auth schema/roles ไม่ใช่ hosted Supabase
bash integration/postgres.sh
```

Tests ใช้ `httptest`, memory repository และ stub repository ส่วน lifecycle tests
เปิด HTTP listener เฉพาะ loopback ด้วย port ที่ระบบเลือก ไม่ติดต่อบริการภายนอก
ไม่ต้องใช้ `.env`, Supabase, ฐานข้อมูล หรือ secret ครอบคลุม:

- response คง 6 product fields เดิม พร้อม specs ที่เกี่ยวข้อง และราคาเป็นจำนวนเต็มเงินบาท
- RAM ram_model_id และแพ็ก 1/2 โมดูลจาก GET ใช้กับ POST validate ได้โดยไม่ hardcode mapping
- ไม่แสดง inactive ทั้งรายการ รายละเอียด ผลค้น และจำนวนรวม
- category/search, pagination, stable order, รายการว่างและหน้าเกินจริง
- UUID, query ผิดรูปแบบ, duplicate parameters และ integer overflow
- error envelope สำหรับ 400/404/405/500 และการไม่รั่วรายละเอียด repository
- CORS ตาม configured origins และ preflight
- POST build validation: input/errors, CPU/RAM mismatch, แพ็ก 1/2 โมดูล และ JSON examples จาก fixture
- local fixture เปิดได้เฉพาะเมื่อตั้งค่าครบ, ค่าเริ่มต้นว่าง และการปฏิเสธ fixture ที่ผิดรูปแบบ
- graceful shutdown รอ active request ให้เสร็จ และปิด connection เมื่อเกิน drain deadline

เพิ่ม auth/profile ownership, role escalation, Member/Admin, CRUD และ DB RPC tests แล้ว
PostgreSQL integration แยกจาก unit tests; ทดสอบ trigger/grants/CRUD/snapshot/rollback ด้วย schema จำลอง
ยังต้องทำ hosted Supabase/Auth และ Render smoke tests ตามคู่มือก่อนส่งงานจริง

## TODO / ข้อตกลงที่รอระยะถัดไป

- Catalog คง `id`, `name`, `category`, `brand`, `price_thb`, `is_active` และเพิ่มเฉพาะ metadata ที่ตกลงแล้ว
- ตกลงชื่อ `specs.cpu_socket`, `specs.ram_type`, `specs.modules_per_package`
  และ `ram_model_id` ระดับเดียวกับ `id` แล้ว และเพิ่มลง Catalog list/detail ตามประเภทสินค้า
- Schema/repository/compatibility/RAM selection ภายในและ open decisions ดู [PHASE2.md](PHASE2.md)
- ต้องตกลงชุดค่า category ที่ Frontend จะใช้; ค่า `cpu` ใน tests เป็น fixture ไม่ใช่ enum ใหม่
- หากต้องการ search ใน brand หรือ fields อื่นเพิ่มเติม ให้ตกลงกับ FE ก่อนขยายพฤติกรรม
- เพิ่ม typed nullable GPU/Storage และ motherboard metadata ตาม [ช่วง A](GPU_STORAGE_A.md) แล้ว
  เพิ่ม direct SATA allocation ตาม [B1](STORAGE_B1.md) แล้ว; ยังไม่มี filters ใหม่,
  GPU/M.2/PSU/Case checks หรือ typed PSU/Case/cooler
- ใช้ server-only Data API RPC, default-deny direct table access และ RLS ตาม [SUPABASE_SETUP.md](SUPABASE_SETUP.md)
- เพิ่ม Auth/Profile/Admin และ private draft persistence/tests ใน branch นี้แล้ว; ยังไม่มี public publication, recovery/OAuth/MFA UI
- ยังต้องตั้ง project/keys/CORS และปิด Confirm email และทดสอบบริการจริงก่อน deploy; ไม่มี privileged key ใน Frontend
