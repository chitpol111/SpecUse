# Guest / Member / Admin และ CRUD สินค้า

ดู request/response contract ใน [AUTH_API.md](AUTH_API.md); ต่อจาก [Supabase setup](SUPABASE_SETUP.md)
รัน migration 005 หลัง 001–004 และ 008 หลัง 005 ด้วย owner SQL connection บน local/test project ก่อน

Guest ใช้ GET Catalog และ POST Validate แบบเดิมได้; Member ใช้ profile/password ของตนเอง
Admin endpoints ตรวจ Bearer token กับ Supabase แล้วอ่าน profiles.role จากฐานข้อมูลทุกครั้ง
ไม่มี/ใช้ token ไม่ได้ 401; role ไม่ใช่ admin 403 ก่อนอ่าน inactive catalog หรือเขียนข้อมูล
ไม่ใช้ role จาก JWT user_metadata/app_metadata หรือ request เป็นสิทธิ์ app
SQL mutation ตรวจ p_actor กับ profiles.role ซ้ำ เพื่อกัน role เปลี่ยนระหว่างคำขอ

## เตรียมบัญชี Admin ล่วงหน้าแบบควบคุม

1. เจ้าของ project เตรียมบัญชีเฉพาะที่ตนควบคุมก่อน demo/deploy; สร้างรหัสผ่านไม่ซ้ำและเก็บใน password manager
   ปิด Confirm email ตาม SUPABASE_SETUP.md แล้วสมัครผ่าน Go Register ด้วย email/password/display_name เท่านั้น
   ต้องได้ 201 พร้อม session; GET /me ยืนยันว่าเริ่ม role=member เสมอ ไม่ต้อง OTP/อีเมลยืนยัน
   อย่าส่ง role, user_metadata หรือ app_metadata และไม่มี /auth/register-admin
2. เจ้าของตรวจ UUID/email ให้ตรงใน Auth Dashboard และ profiles ของบัญชีนี้
   อย่ายกระดับตามชื่อ display_name หรือ email ที่ส่งมาใน request และไม่พิมพ์ token/password ลง logs
3. ใช้ Supabase SQL Editor ที่ลงชื่อเข้าใช้เป็นเจ้าของ หรือ owner DB connection เท่านั้น
   แทน UUID/email placeholder ด้านล่างด้วยบัญชีที่ตรวจแล้วก่อนรัน (ไม่ต้องใส่รหัสผ่านหรือ key ใน SQL):

```sql
UPDATE public.profiles p SET role='admin',updated_at=now()
FROM auth.users u
WHERE p.id=u.id
  AND u.id='REPLACE_WITH_VERIFIED_USER_UUID'::uuid
  AND u.email='REPLACE_WITH_VERIFIED_EMAIL'
  AND p.role='member'
RETURNING p.id,p.role;
```

ต้องได้หนึ่งแถว role=admin; หากไม่มีแถวหรือเกินหนึ่งแถวให้หยุดตรวจ UUID/email/role ก่อน ไม่ขยาย WHERE ให้กว้าง
4. Login ด้วยบัญชีที่เตรียมไว้ → GET /me ต้อง role=admin และ GET /admin/products ต้อง 200
   ใช้บัญชี member อีกบัญชีตรวจ 403; ไม่มี Bearer ต้อง 401 ก่อนนำไป demo
5. เจ้าของส่งมอบ credentials ผ่าน password manager/ช่องทางที่ควบคุมได้เท่านั้น ไม่ส่งใน PR/แชต/source
   ไม่ใช้บัญชี Admin เป็นข้อมูลตัวอย่างและไม่สร้างบัญชีจริงผ่าน unit tests หรือ migration

อย่าใช้ service key ผ่าน Frontend และไม่มี public endpoint ยกระดับสิทธิ์
public Register สร้าง member เสมอ; SQL owner promotion เป็นขั้นตอนดูแลระบบล่วงหน้าแยกจาก public API
ลดสิทธิ์ด้วย owner SQL เช่นเดียวกัน; คำขอถัดไปอ่าน role ใหม่ไม่รอ JWT หมดอายุ
ไม่สร้าง admin/email/password ใน seed; tests ใช้บัญชีจำลองใน disposable DB เท่านั้น

## Product operations

- GET /api/v1/admin/products: pagination/filter แบบ catalog แต่รวม inactive
- GET รายการมี `revision_format: "sha256"` แม้ไม่มีสินค้า และแต่ละรายการรวมถึงผล
  POST/PATCH มี `revision` ของสินค้าสำหรับใช้ตรวจข้อมูลเก่า
- POST /api/v1/admin/products: name/category/brand/price_thb/is_active ต้องครบ; server สร้าง UUID
- PATCH /api/v1/admin/products/{id}: sparse name/brand/price_thb/is_active/specs/ram_model_id
  ต้องส่ง `expected_revision` ที่ได้จากสินค้าที่เปิดแก้เสมอ; category immutable;
  specs replace ทั้ง object; id/role ไม่รับจาก request
- DELETE /api/v1/admin/products/{id}: 204 และ is_active=false เสมอ ไม่ลบแถว/metadata เพื่อรักษา references
- CPU/MB/RAM ต้องมี typed metadata ที่จำเป็น; RAM model ต้อง provision ใน DB อย่างควบคุมก่อน
- nullable GPU/Storage metadata ตาม contract เดิม; synthetic evidence ถูกปฏิเสธก่อนเขียนผ่าน DB adapter
- Guest เห็นเฉพาะ active; Validate ใช้ราคาปัจจุบันจาก snapshot ที่อ่านจาก DB หลัง mutation
- Postgres transaction เขียน products และ typed rows พร้อมกัน; FK/constraint error rollback ทั้ง mutation
- PATCH ตอบ 400 เมื่อไม่มี `expected_revision` หรือรูปแบบไม่ถูกต้อง และตอบ 409 หาก
  ข้อมูลสินค้าที่เปิดแก้ไม่ตรงกับข้อมูลปัจจุบัน หรือเปลี่ยนระหว่างคำขอ หลัง 409 ให้เปิด
  รายการใหม่แล้วตรวจข้อมูลก่อนแก้ซ้ำ; ห้าม retry payload เก่าอัตโนมัติ
- `revision` เป็น fingerprint ของข้อมูลสินค้าที่ API ส่ง ไม่ใช่เลขลำดับหรือ secret;
  หากข้อมูลเปลี่ยนแล้วกลับมาเหมือนเดิม fingerprint อาจเหมือนเดิม
- DELETE สำหรับปิดใช้งานยังไม่ตรวจ revision จากหน้าจอ ควรประสานผู้ดูแลก่อนปิดสินค้า

### การ deploy รุ่นตรวจ revision

ต้องอัปเดต Backend ก่อน Frontend และงดแก้สินค้าในช่วงเปลี่ยนรุ่น: หน้า Admin รุ่นเก่า
ไม่ส่ง `expected_revision` จึง PATCH ไม่ได้กับ Backend ใหม่ และ Frontend ใหม่ไม่สามารถ
แก้สินค้ากับ Backend เก่าได้ การดู catalog และ member builds ไม่เปลี่ยน contract
หลัง deploy ทั้งสองส่วนให้ตรวจการแก้ไขจากหน้าค้างสองหน้ากับสินค้าทดสอบที่ได้รับอนุญาต
โดยให้คำขอที่สองได้ 409 ก่อนกลับมาแก้สินค้าใช้งานจริง การ rollback Backend ไปเวอร์ชัน
เดิมจะนำข้อจำกัดการเขียนทับกลับมา จึงต้องงดแก้สินค้าระหว่าง rollback และตรวจคู่เวอร์ชัน

Backend รุ่นนี้ต้องติดตั้ง migration 008 หลัง migrations 001–005 ก่อน deploy; migration 005 เดิมไม่ถูกแก้
ฐานที่ติดตั้ง 005 แล้วต้องลง 008 แยก ไม่รัน 005 ซ้ำ และตรวจ release plan ก่อนเปลี่ยนฐานจริง
RPC เดิมยังคงอยู่เพื่อให้ Backend รุ่นเดิมและ rollback ใช้ได้ แต่รุ่นเดิมไม่มีการตรวจ concurrency นี้
ต้องหลีกเลี่ยงการแก้สินค้าระหว่าง deploy/rollback ที่มี Backend รุ่นเก่าใช้งาน

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
