# Supabase + สมาชิก — การตั้งค่าและขอบเขตผลทดสอบ

Contract ที่ Frontend ใช้: [AUTH_API.md](AUTH_API.md); React เรียก Go เท่านั้น
Go ใช้ net/http กับ Supabase Auth และ Data API RPC ไม่มี dependency ของ SDK ภายนอก
Catalog/Validate ใช้ repository interfaces เดิม และเรียก backend_catalog_snapshot หนึ่งครั้งต่อคำขอ
SQL function อ่านทุกตารางใน SQL statement เดียว ไม่ fetch ตารางแยกจนราคา/specs คนละ snapshot

## สิ่งที่เจ้าของบัญชีต้องเตรียม

ยังไม่มี SUPABASE_* หรือบัญชี Render ใน environment นี้ จึงยังไม่ทดสอบ hosted Supabase/อีเมลจริง/deploy
ตั้งค่าผ่าน Supabase Dashboard และ backend environment/secret manager ไม่ส่ง secrets ในแชต

1. เลือก project local/test ที่เจ้าของควบคุม; ไม่สร้าง project/บริการแบบมีค่าใช้จ่ายอัตโนมัติ
2. Authentication → Email เปิด provider และ Confirm email; minimum password length >=8
3. Confirm signup template ใช้ {{ .Token }} ให้กรอก OTP ผ่าน POST /api/v1/auth/confirm
   ตั้ง SMTP ที่ทดสอบส่งได้และทดสอบกับอีเมลตนเอง; delivery/rate limit ขึ้นกับ provider settings
4. Settings → API Keys: project URL, publishable key และ secret key สำหรับ server
   legacy anon/service_role JWT keys ใช้ได้เช่นกัน แต่ไม่ใส่ server key ใน Frontend
5. เปิด Data API สำหรับ public schema เพื่อให้ Go เรียก RPC; anonymous/member ไม่มีสิทธิ์ table/RPC ที่สร้างในงานนี้
6. รัน migrations 001,002,003,004,005,006 ด้วย project-owner SQL connection/SQL Editor ตามลำดับ
   ถ้า 001/002 เคยรันแล้ว อย่ารันซ้ำ; ตรวจ migration history และ backup ก่อนใช้ฐานที่มีข้อมูล
   startup ไม่รัน migrations หรือ seed ให้อัตโนมัติ

Environment ของ Go:

```bash
export APP_ENV=local
export CATALOG_BACKEND=supabase
export SUPABASE_URL=https://YOUR-PROJECT.supabase.co
export SUPABASE_PUBLISHABLE_KEY=SET_VIA_SECRET_MANAGER
export SUPABASE_SECRET_KEY=SET_VIA_SECRET_MANAGER
export SUPABASE_JWT_ISSUER=https://YOUR-PROJECT.supabase.co/auth/v1
export SUPABASE_JWT_AUDIENCE=authenticated
export CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
export PORT=8080
cd backend
go run ./cmd/api
```

แทนค่าตัวอย่างผ่านช่องทางตั้งค่าที่เหมาะสม อย่าคัดลอก key จริงลง shell history หรือไฟล์ที่ Git track
ใช้ http ได้เฉพาะ APP_ENV=local และ host localhost/127.0.0.1 ของ Supabase local stack
issuer/audience ต้องตรง project จริง ไม่ตั้งจากค่าที่ client ส่ง
ไม่ตั้ง LOCAL_CATALOG_FILE ร่วมกับ Supabase mode: configuration แบบผสมถูกปฏิเสธ
หาก provider/DB ล่มจะตอบ error ไม่ fallback ไป fixture
memory mode สำหรับ tests/local เดิมยังมี; ถ้าตั้ง SUPABASE_URL ต้องตั้ง CATALOG_BACKEND=supabase ชัดเจน
ยังไม่ใช่หลักฐานว่า hosted connection สำเร็จจนเจ้าของรัน checklist ด้านล่าง

## Schema / permissions

- 003 สร้าง profiles และ trigger จาก auth.users: role='member' เสมอ ไม่อ่าน role จาก metadata
- โปรไฟล์ของบัญชีเก่าถูกเติมแบบ member เฉพาะที่ยังไม่มี ไม่ overwrite role เดิม
- GET/PATCH profile ใช้ UUID จาก token ที่ Supabase verified แล้ว ไม่รับ owner/role/email จาก body
- 004 เพิ่ม hardware_specs แบบ nullable บน products สำหรับ GPU/Storage และ metadata เพิ่มของ motherboard
  CPU/socket, motherboard/socket+RAM และ RAM models/packages ยังคง typed tables เดิม
- ไม่มีการ backfill metadata หรือ synthetic hardware evidence ให้สินค้าใน DB
- เปิด RLS และ revoke table privileges จาก anon/authenticated/service_role; ไม่มี client policies
- 005 เพิ่ม Admin mutation RPC ตรวจ actor role ซ้ำ, 006 เพิ่ม backend_health RPC สำหรับ readiness
- เฉพาะ service_role เรียก backend_catalog_snapshot/backend_profile_get/backend_profile_update และ Admin/health RPC ได้
  ฟังก์ชัน SECURITY DEFINER มี search_path='' และใช้ชื่อ schema ชัดเจน
- Go ตรวจ token ทุกครั้งผ่าน Auth GET /user แล้วตรวจ iss/aud/exp/nbf/sub และเทียบ user UUID
  SQL role ที่ app ใช้อ่าน profiles เป็นฝั่ง server; token role/user_metadata ไม่กำหนดสิทธิ์ app
- ไม่เก็บ password/hash/refresh token เอง; session เป็นของ Supabase และไม่มี token logging

## Development seed

seeds/development.sql เป็นข้อมูล DEMO case ที่ไม่มี hardware specs/evidence ไม่ใช่สินค้าหรือราคาจริง
ตั้ง app.allow_development_seed ผ่าน owner SQL session แล้วจึงรันไฟล์ ไม่มี seed จาก startup

```sql
SELECT set_config('app.allow_development_seed','on',false);
-- psql: \i seeds/development.sql
```

ไม่มีข้อมูล 11500/synthetic hardware ถูกนำไปเป็น production catalog: ตัวอย่าง 11500 อยู่ใน test fixtures เดิม
เพิ่มสินค้าจริงต้องมีข้อมูลและหลักฐานที่ตรวจรับแล้ว; seed ไม่สร้าง admin หรือบัญชี Auth

## ทดลอง flow กับ project ที่ตั้งค่าแล้ว

1. POST /auth/register ด้วย email/password/display_name → 201, session=null เมื่อ confirmation เปิด
2. รับ OTP จากอีเมล → POST /auth/confirm ด้วย email/token → session
3. GET /me ด้วย Bearer access token → profile role=member
4. PATCH /me เฉพาะ display_name → ชื่อเปลี่ยน; role/id/email ใน body ต้อง 400
5. POST /auth/refresh ด้วย refresh_token → แทนทั้ง access/refresh pair
6. PATCH /me/password ด้วย current_password/new_password → แทน session ใหม่ที่ reauthenticate
7. POST /auth/logout ด้วย Bearer ไม่มี body → 204 แล้วล้าง token ทั้งคู่ใน FE
8. Refresh หลัง logout ต้องใช้ไม่ได้; access JWT เก่าอาจใช้ได้จน exp จึงห้ามกล่าวว่า revoke JWT ทันที
9. GET /products และ POST /builds/validate ใช้ DB snapshot จริง; ตรวจราคา/active/metadata ที่ provision เอง

หน้านี้ยังไม่ให้ผล PASS ของ hosted checklist ต้องบันทึก project/test date และผลจริงแยกเมื่อรัน

## Tests ที่ไม่ใช้ production Supabase

```bash
cd backend
go test ./...
go vet ./...
```

unit/API tests ใช้ provider stubs และ httptest Auth/Data API ทดสอบ token ผิด/หมดอายุ/issuer/audience,
role จาก DB, ownership, role escalation, refresh/session fields, upstream errors และ 11500 regression
fake Auth server ไม่ใช่ Supabase จริง จึงไม่ถือเป็นผลทดสอบ SMTP/JWT signing ของ hosted project
migrations ตรวจเพิ่มได้กับ PostgreSQL container และ auth.users/roles จำลอง; ผลนี้ไม่แทน hosted Supabase integration

Auth session/logout และ privileged-key semantics อ้างอิงเอกสารทางการใน AUTH_API.md

Render configuration และ hosted smoke checklist ดู [RENDER.md](RENDER.md)
`GET /api/v1/health` เช็ก DB RPC ภายใน 2 วินาทีใน Supabase mode; ไม่ใช่หลักฐานว่า Auth/SMTP ใช้งานได้ครบ
