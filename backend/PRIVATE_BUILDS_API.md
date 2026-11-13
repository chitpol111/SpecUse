# Member private drafts — API contract และวิธีทดสอบ

เริ่มจาก `a303f7c` ใน branch/worktree แยก `feat/backend-member-builds` ตามที่ยืนยัน 2026-11-12
Draft PR #12 retarget ไป `main` หลัง PR #11 รวมแล้ว; private draft ยังไม่ merge หรือ deploy
Render ใช้ `feat/backend-render-ready`; ห้าม push งานนี้ไป branch นั้นโดยไม่ประสาน
เจ้าของรายงานว่า Auto-Deploy ปิดแล้ว ให้ตรวจอีกครั้งก่อน release
ผล hosted Auth/Admin/Catalog ก่อนหน้านี้เป็นผลที่เจ้าของรายงาน; รอบ private draft นี้ยังทดสอบเฉพาะ local/จำลอง

## ขอบเขตที่ยืนยัน

- บันทึก private draft ที่ยังไม่ครบหรือมี compatibility conflict ได้ รวม inactive ถ้า reference ยังอยู่ใน DB
- schema เตรียม public/private แต่ public writes ตอบ 409; ยังไม่มี public list/detail/share หรือเกณฑ์เผยแพร่
- บันทึก IDs และจำนวนตามจริงทุกแถว รวมหลาย SKU ในหมวดเดียวและหลาย RAM models ไม่ลดเหลือหมวดละหนึ่ง
- Backend ตรวจ Supabase token/expiry/issuer/audience และ profile role ตามเดิม; member/admin เข้าถึงเฉพาะของตนเอง
- ไม่รับ owner/role/ราคา/ยอดรวม/ผลตรวจจาก client ไม่เก็บผล validate หรือราคาล็อกตอนบันทึก
- Catalog/validate เดิม รวม legacy 11500, GPU/Storage/B1 และ CORS คงเดิม ไม่เพิ่ม hardware checks

## Endpoints

Base local `http://127.0.0.1:8080/api/v1`; ต้องใช้ code ใน branch ใหม่และ test Supabase ที่มี migrations 001–008
ทุก endpoint ส่ง `Authorization: Bearer <access_token>` และ responses ใช้ `Cache-Control: no-store`
ใช้ session/refresh ผ่าน `protectedRequest` ภายใน `createAuthClient` เดิมของ FE ไม่สร้าง Auth flow อีกชุด

| Method / path | Success |
|---|---|
| GET /me/builds?page=1&page_size=20 | 200 data:{items,page,page_size,total}; ของเจ้าของ token เท่านั้น |
| POST /me/builds | 201 data:{build} |
| GET /me/builds/{id} | 200 data:{build} |
| PATCH /me/builds/{id} | 200 data:{build} |
| DELETE /me/builds/{id} | 204 ไม่มี body; ลบ Build/รายการลูก ไม่ลบสินค้า |

List items เป็น `{id,name,description,visibility,created_at,updated_at}` ไม่มีราคา/รายการอุปกรณ์
เรียง updated_at ล่าสุดก่อน แล้ว UUID ascending; หน้าเกินจริงได้ items:[] แต่ total คงเดิม
page/page_size เป็นจำนวนเต็มบวกตาม catalog และต้องไม่ทำให้ offset overflow; ไม่รับ query owner_id/filters อื่น
ค่าเริ่มต้น page=1/page_size=20; ไม่ส่งค่าว่างหรือซ้ำ

## Write request

POST ต้องมี name/items/ram_items; description default `""`, visibility default `"private"`
name trim แล้ว 1–100 Unicode characters; description เก็บรูปแบบข้อความเดิม ไม่เกิน 2000 characters; ไม่รับ NUL/null
items/ram_items เป็น arrays ที่ไม่เป็น null, ว่าง [] ได้; body ไม่เกิน 1 MiB และ Content-Type: application/json

```json
{
  "name":"ร่างสเปกของฉัน",
  "description":"ยังเลือกอุปกรณ์ไม่ครบ",
  "visibility":"private",
  "items":[],
  "ram_items":[]
}
```

- items: `{product_id,quantity,owned_quantity}` สำหรับ non-RAM
- ram_items: `{ram_model_id,required_modules,owned_modules}`; model ID ไม่ใช่ SKU/แพ็ก RAM
- จำนวนใช้ integer int64 แบบเดียวกับ domain เดิม: required/quantity>=1, 0<=owned<=required/quantity
- owned เป็นส่วนหนึ่งของจำนวนที่ต้องการ ไม่บวกซ้ำ ไม่ลดจำนวนที่เก็บเหลือเฉพาะยอดซื้อเพิ่ม
- UUID canonical (รับ hex ตัวใหญ่แล้ว normalize lower), ไม่รับ product/model ซ้ำแบบไม่แยกตัวพิมพ์
- RAM SKU ใน items ตอบ 400; reference ไม่มีอยู่ 404; inactive และ model ที่ไม่มีแพ็ก active ยังเก็บ draft ได้
- ไม่รับ id/owner_id/role/price_thb/total_price_thb/compatible/summary/availability ใน write body

PATCH เปลี่ยนเฉพาะ fields ที่ส่งมา อย่างน้อยหนึ่ง field; null ไม่ใช่การละ field และถูกปฏิเสธ
`{"name":"ชื่อใหม่"}` คงอุปกรณ์ทุกแถว; `items` หรือ `ram_items` ที่ส่งมา **แทนทั้ง array นั้น**
[] ล้าง array นั้นโดยเจตนา; array ที่ไม่ได้ส่งคงเดิมใน DB transaction ไม่ถูกสร้างใหม่จากข้อมูล GET ที่ล้าสมัย
หาก request ใดผิด/public/ไม่มีสิทธิ์ ต้องไม่บันทึกบางส่วน

## Read response และ availability

Build มี fields ใน write พร้อม id/created_at/updated_at ที่ server สร้าง และ read-only:

- items[].product_summary: `{name,category,is_active}` จาก catalog ปัจจุบัน รวมสินค้าที่ inactive
- ram_items[].ram_model_summary: `{ram_type}` จาก model จริง ไม่มีชื่อ/model active flag ที่เดาขึ้น
- availability: `{ready_for_validation,issues:[{code,message,product_id,ram_model_id}]}`

issue ใช้ keys เหมือนกันทุกแถว ID ที่ไม่เกี่ยวข้องเป็น null; ไม่มีปัญหาส่ง issues:[]
PRODUCT_INACTIVE ใช้ product_id แม้ owned_quantity=quantity เพราะ validate เดิมไม่รับ inactive non-RAM
RAM_PACKAGES_UNAVAILABLE ใช้ ram_model_id เฉพาะ required_modules>owned_modules และไม่มีแพ็ก active ของ model
RAM ที่มีครบแล้วไม่บล็อกเพราะไม่มีแพ็กขาย ตรงกับ validate เดิม
active package count เป็นข้อมูลภายใน ไม่ส่งเป็น public JSON field

ready_for_validation=true หมายถึงผ่านการตรวจความพร้อมของ catalog สำหรับเงื่อนไขข้างต้นเท่านั้น
ยังต้องเรียก POST /builds/validate ด้วย IDs/counts เดิมเพื่อคิดราคาและตรวจ compatibility
ไม่ได้หมายถึง compatible/ครบทุกหมวด/อนุญาต public และไม่ได้รับประกันว่า validate จะไม่มี error อื่น
GET/POST/PATCH saved Build ไม่คำนวณหรือคืน total_price_thb จึงไม่แทนราคาที่ไม่ทราบด้วย 0
availability false หรือ validate error ต้องล้างผลราคาที่ล้าสมัยและแสดง “ยังตรวจราคาไม่ได้” ไม่ลบรายการ
summary/availability อ่านใหม่จาก DB; การ deactivate/เปลี่ยนแพ็กหลัง save มีผลเมื่ออ่านครั้งถัดไป

ดู [unavailable.response.json](examples/drafts/unavailable.response.json) สำหรับตัวอย่างเต็ม
ข้อมูลในตัวอย่างเป็น synthetic เพื่อแสดง JSON เท่านั้น ไม่ใช่ SKU/specs/evidence ของสินค้าจริง

## การรักษาข้อมูลบน FE ที่เลือกได้หมวดละหนึ่ง

1. โหลด detail แล้วเก็บ items/ram_items **ฉบับเต็ม** แยกจากค่าช่องเลือกในหน้าปัจจุบัน
2. อย่าแปลง array เป็น object ที่มี category เป็น key เพราะหลาย SKU จะถูกทับ
3. ห้ามเอา requestPayload ของหน้าจัดสเปกที่มีหมวดละหนึ่งไป PATCH ทั้ง array ของ draft ที่มีข้อมูลมากกว่า
4. แก้ชื่อ/description อย่างเดียวส่งเฉพาะ fields นั้น ไม่ส่ง arrays ช่วยรักษาข้อมูลที่ UI ยังแก้ไม่ได้
5. ถ้าจะแก้อุปกรณ์ ให้แก้ row ตาม product_id/ram_model_id ในฉบับเต็ม แล้ว serialize ทุกแถวกลับโดยเลือกเฉพาะ write fields
6. ถ้า UI แทนหลาย SKU/หลาย model/quantity>1 หรือจำนวนที่ไม่ใช่ JS safe integer ไม่ได้ ให้แสดงรายการครบ/แจ้งข้อจำกัด
   และปิดการแก้อุปกรณ์ของ draft นั้นชั่วคราว ยังคงแก้ metadata ได้ ห้าม reset จำนวนเป็น 1 หรือเลือกแถวแรกแล้ว save ทับ
7. เอารายการออกเมื่อผู้ใช้สั่งชัดเจนเท่านั้น รวม inactive/ไม่มีแพ็ก active ไม่ลบทิ้งตอน load, filter หรือ validate ล้มเหลว

ตัวอย่างสร้าง write arrays จาก state ฉบับเต็ม (ตัวอย่างในเอกสาร ไม่ได้แก้ Frontend):

```js
const body = {
  items: fullDraft.items.map(({ product_id, quantity, owned_quantity }) =>
    ({ product_id, quantity, owned_quantity })),
  ram_items: fullDraft.ram_items.map(({ ram_model_id, required_modules, owned_modules }) =>
    ({ ram_model_id, required_modules, owned_modules })),
};
```

Backend แยกไม่ได้ว่าแถวที่หายจาก replacement array เป็นการลบโดยเจตนาหรือข้อจำกัด UI
tests รอบนี้ยืนยัน backend roundtrip/metadata patch/full-array replacement; ยังไม่ได้ทดสอบร่วมกับ FE private draft ล่าสุด

## Errors

คง `{"error":{"code":"...","message":"..."}}` ทุก endpoint

| HTTP / code | กรณี |
|---|---|
| 400 VALIDATION_ERROR | body/fields/counts/UUID/duplicates/query ผิด, ส่ง read-only fields |
| 401 UNAUTHORIZED | ไม่มี/invalid/expired token |
| 403 FORBIDDEN | identity/role ใช้ไม่ได้; Admin ไม่ได้สิทธิ์พิเศษเหนือ ownership |
| 404 NOT_FOUND | reference ไม่มี หรือ Build ไม่มี/ไม่ใช่ของตนเอง (ไม่เปิดเผยว่าเป็นของคนอื่น) |
| 409 PUBLIC_BUILD_NOT_ENABLED | visibility=public ก่อนมี publication policy |
| 409 CONFLICT | database reference/constraint conflict |
| 405 / 413 / 415 | method, body ใหญ่เกิน 1 MiB, content type |
| 429 / 503 / 500 | rate limit, provider/DB/RPC ไม่พร้อม, internal error ตาม envelope เดิม |

## Schema / migration / permissions

007_member_builds.up.sql สร้าง builds/build_items/build_ram_items, owner nullable FK profiles,
รายการอุปกรณ์ FK แบบ restrict, counts checks และ position เพื่อคงลำดับและทุกแถว
DELETE Build cascade รายการลูก; Auth account deletion → profiles cascade → owner=NULL และ private โดยเก็บรายการไว้
เปิด RLS และ revoke direct table privileges รวม service_role; ไม่มี browser policies
RPC list/get/create/patch/delete เฉพาะ service_role และตรวจ actor/owner ซ้ำใน DB; helpers เรียกตรงไม่ได้
create/patch เป็น transaction; PATCH lock row ก่อนแก้ ไม่เขียนทับ fields/arrays ที่ omitted
ไม่เก็บ password/token/ราคา/specs ที่ client ส่ง และไม่เพิ่ม seed สินค้าหรือ hardware evidence

**ยังไม่ apply 007 กับ specuse-dev** ซึ่งกำลังใช้งานจริงและรัน 001–006 แล้ว
ก่อน apply ต้องตรวจ migration history, table/function names และ schema จริง ประสานเจ้าของ และทดสอบฐานแยกก่อน
ไม่รัน 001–006 ซ้ำ ไม่มี migration/seed อัตโนมัติเมื่อ server เริ่ม

ผลรันจริงดู [PRIVATE_BUILDS_VERIFICATION.md](PRIVATE_BUILDS_VERIFICATION.md)

คู่มือทดสอบร่วม FE กับ Supabase local แยก: [PRIVATE_BUILDS_E2E.md](PRIVATE_BUILDS_E2E.md)

## รันและ curl

ทดสอบโดยไม่ใช้ Supabase จริงจาก backend/:

```bash
go test ./...
go vet ./...
go test -race ./internal/...
bash integration/postgres.sh
```

SQL integration ต้องมี Docker ใช้ PostgreSQL ชั่วคราวพร้อม Auth schema/roles จำลอง แล้วลบ container
ตรวจ migrations 001–008 ในฐานว่างชั่วคราว ไม่ใช่คำสั่งสำหรับรันซ้ำบน project จริง
ไม่ใช้ synthetic examples เป็นข้อมูล specuse-dev

Manual API test: ใช้ **test Supabase ที่ตั้งค่าและมี 001–008 แล้ว** ตาม SUPABASE_SETUP.md
export URL/keys ผ่าน secret manager, ไม่ตั้ง LOCAL_CATALOG_FILE, ไม่ส่ง keys ในแชต/Git

```bash
CATALOG_BACKEND=supabase APP_ENV=local PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

อีก terminal ตั้ง ACCESS_TOKEN จาก Go login ของบัญชีทดสอบอย่างปลอดภัย ไม่ echo/ใช้ shell tracing
คำสั่งด้านล่างต้องมี curl ที่รองรับ --fail-with-body และ Python 3; เก็บ responses ส่วนตัวนอก Git

```bash
set +x
API_BASE=http://127.0.0.1:8080/api/v1
DRAFT_TEST_DIR=$(mktemp -d)
chmod 700 "$DRAFT_TEST_DIR"
# 201: draft ว่าง ไม่ต้องมีข้อมูลสินค้าทดสอบใน DB
curl --silent --show-error --fail-with-body -w 'HTTP %{http_code}\n' \
  -o "$DRAFT_TEST_DIR/create.json" -X POST "$API_BASE/me/builds" \
  -H "Authorization: Bearer $ACCESS_TOKEN" -H 'Content-Type: application/json' \
  --data-binary @examples/drafts/create-empty.request.json
BUILD_ID=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["data"]["build"]["id"])' "$DRAFT_TEST_DIR/create.json")
# 200: list/detail/metadata patch
curl --silent --show-error --fail-with-body -o "$DRAFT_TEST_DIR/list.json" \
  "$API_BASE/me/builds?page=1&page_size=20" -H "Authorization: Bearer $ACCESS_TOKEN"
curl --silent --show-error --fail-with-body -o "$DRAFT_TEST_DIR/detail.json" \
  "$API_BASE/me/builds/$BUILD_ID" -H "Authorization: Bearer $ACCESS_TOKEN"
curl --silent --show-error --fail-with-body -o "$DRAFT_TEST_DIR/patch.json" \
  -X PATCH "$API_BASE/me/builds/$BUILD_ID" -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' --data-binary @examples/drafts/patch-metadata.request.json
# 204: ไม่มี body
curl --silent --show-error --fail-with-body -w 'HTTP %{http_code}\n' -o /dev/null \
  -X DELETE "$API_BASE/me/builds/$BUILD_ID" -H "Authorization: Bearer $ACCESS_TOKEN"
```

ตรวจ status แต่ละขั้นก่อนทำต่อ ล้าง token variables/private temp files เมื่อจบทดสอบ
ตัวอย่าง create-multiple.request.json ใช้ synthetic IDs สำหรับแสดง array เท่านั้น ต้องแทน IDs จริงของ test project ก่อน curl
ตัวอย่าง public-disabled.request.json ควรได้ 409; ตรวจ owner อื่นด้วยบัญชีทดสอบที่สองต้อง 404 แม้เป็น Admin
health 200 เดิมพิสูจน์ backend_health จาก 006 ไม่ได้ยืนยันว่า 007/draft RPC ถูกติดตั้งแล้ว
ก่อนเปิดใช้จริงต้องตกลง PR/base, migration และ deployment แยกกัน โค้ดนี้ยังไม่อยู่บน Render URL ปัจจุบัน
