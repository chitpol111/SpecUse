# Guest / Member / Admin และ CRUD สินค้า

ดู request/response contract ใน [AUTH_API.md](AUTH_API.md); ต่อจาก [Supabase setup](SUPABASE_SETUP.md)
รัน migration 005 หลัง 001–004 ด้วย owner SQL connection บน local/test project ก่อน

Guest ใช้ GET Catalog และ POST Validate แบบเดิมได้; Member ใช้ profile/password ของตนเอง
Admin endpoints ตรวจ Bearer token กับ Supabase แล้วอ่าน profiles.role จากฐานข้อมูลทุกครั้ง
ไม่มี/ใช้ token ไม่ได้ 401; role ไม่ใช่ admin 403 ก่อนอ่าน inactive catalog หรือเขียนข้อมูล
ไม่ใช้ role จาก JWT user_metadata/app_metadata หรือ request เป็นสิทธิ์ app
SQL mutation ตรวจ p_actor กับ profiles.role ซ้ำ เพื่อกัน role เปลี่ยนระหว่างคำขอ

## Provision Admin แบบควบคุม

1. สมัคร/ยืนยันอีเมลผ่าน flow ปกติ จะได้ member เสมอ
2. เจ้าของ project ตรวจ UUID และ email ของบัญชีใน Auth Dashboard
3. ใช้ SQL Editor/owner connection เท่านั้น ตรวจให้ตรงทั้ง UUID/email ก่อนเปลี่ยน role:

```sql
UPDATE public.profiles p SET role='admin',updated_at=now()
FROM auth.users u
WHERE p.id=u.id
  AND u.id='REPLACE_WITH_VERIFIED_USER_UUID'::uuid
  AND u.email='REPLACE_WITH_VERIFIED_EMAIL';
```

ต้องได้ UPDATE 1; อย่าใช้ service key ผ่าน Frontend และไม่มี public endpoint ยกระดับสิทธิ์
ลดสิทธิ์ด้วย owner SQL เช่นเดียวกัน; คำขอถัดไปอ่าน role ใหม่ไม่รอ JWT หมดอายุ
ไม่สร้าง admin/email/password ใน seed; tests ใช้บัญชีจำลองใน disposable DB เท่านั้น

## Product operations

- GET /api/v1/admin/products: pagination/filter แบบ catalog แต่รวม inactive
- POST /api/v1/admin/products: name/category/brand/price_thb/is_active ต้องครบ; server สร้าง UUID
- PATCH /api/v1/admin/products/{id}: sparse name/brand/price_thb/is_active/specs/ram_model_id
  category immutable ใน milestone นี้; specs replace ทั้ง object; id/role ไม่รับจาก request
- DELETE /api/v1/admin/products/{id}: 204 และ is_active=false เสมอ ไม่ลบแถว/metadata เพื่อรักษา references
- CPU/MB/RAM ต้องมี typed metadata ที่จำเป็น; RAM model ต้อง provision ใน DB อย่างควบคุมก่อน
- nullable GPU/Storage metadata ตาม contract เดิม; synthetic evidence ถูกปฏิเสธก่อนเขียนผ่าน DB adapter
- Guest เห็นเฉพาะ active; Validate ใช้ราคาปัจจุบันจาก snapshot ที่อ่านจาก DB หลัง mutation
- Postgres transaction เขียน products และ typed rows พร้อมกัน; FK/constraint error rollback ทั้ง mutation

## วิธีทดสอบหลังตั้ง Supabase แล้ว

ใช้ access token ของ member/admin จาก Go login โดยไม่พิมพ์ token ลง source หรือ console
แทน API_BASE และ access token ผ่าน environment ของเครื่อง (อย่า commit .env):

```bash
curl -i "$API_BASE/admin/products" -H "Authorization: Bearer $ACCESS_TOKEN"
curl -i -X POST "$API_BASE/admin/products" \
  -H "Authorization: Bearer $ACCESS_TOKEN" -H 'Content-Type: application/json' \
  --data '{"name":"[DEVELOPMENT DEMO ONLY] Case","category":"case","brand":"DEMO ONLY","price_thb":100,"is_active":true}'
```

ทดสอบ member token ต้อง 403, ไม่มี token ต้อง 401; Admin สร้าง/แก้ราคา/ปิดสินค้าแล้วตรวจผ่าน public Catalog/Validate
ตัวอย่าง DEMO ไม่ใช่ข้อมูลสินค้าจริง ต้องลบ/ปิดเมื่อเสร็จการทดสอบ
การเปลี่ยนราคาจาก Admin เป็นการบำรุง catalog โดยผู้มีสิทธิ์; build request ยังส่งราคาเองไม่ได้

## Tests / ข้อจำกัด

```bash
go test ./...
go vet ./...
bash integration/postgres.sh
```

integration script ต้องใช้ Docker และ image postgres:17-alpine; เปิด container local แบบ network none/tmpfs
สร้าง auth schema/roles จำลอง รัน migrations และทดสอบ trigger/grants/CRUD/snapshot/rollback แล้วลบ container
ไม่ใช้ production credentials และไม่ใช่ Supabase Auth/SMTP หรือ hosted Data API integration
ยังต้องทดสอบ project จริงก่อน demo/deploy และยังไม่มี saved-build persistence ใน milestone นี้
