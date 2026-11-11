# Auth / Profile / Admin contract

ฐาน main 000dd9f; ยืนยันค่าเริ่มต้นกับเจ้าของ Backend 2026-11-11
ทุก request จาก React ใช้ Go `/api/v1` เท่านั้น ไม่มี Supabase SDK flow ซ้ำและไม่มี cookies
Go เรียก Supabase Auth / Data API ฝั่ง server; Supabase เป็นผู้เก็บ/ตรวจรหัสผ่านและออก session
สัญญานี้ส่งก่อน implementation แล้ว; ปัจจุบันมีโค้ด/tests ใน branches แบบต่อกัน ยังไม่ merge
ต้องตั้งค่า project และทดสอบบริการจริงตาม SUPABASE_SETUP.md/RENDER.md ก่อนอ้างว่าเชื่อมต่อหรือ deploy สำเร็จ

## ค่าเริ่มต้นและขั้นตอน

- Supabase Email provider เปิด, Confirm email เปิด, minimum password length ตั้งอย่างน้อย 8
- Profile แก้ได้เฉพาะ display_name (trim, 1–100 Unicode characters, ไม่รับ NUL); email/id/role เป็น readonly
- Email template ของ Confirm signup ใช้ `{{ .Token }}` ให้ผู้ใช้กรอก OTP ใน FE แล้ว POST ผ่าน Go
  ไม่ใช้ implicit redirect ที่ส่ง tokens ใน URL; ไม่ต้องทำ OAuth/PKCE flow อีกชุด
- สมัคร → ยืนยัน email ด้วย OTP → ได้ session; login หลังยืนยันแล้วใช้ email/password ผ่าน Go
- OTP เป็น string ตัวเลข 6–10 หลักตาม project configuration; อายุ/rate limits ใช้ค่าของ Supabase
- Login/Confirm/Refresh/เปลี่ยน password ส่ง session ที่ Supabase ออกให้; FE เปลี่ยน token pair ทั้งชุดแบบ atomic
- FE เก็บ token pair ใน memory; หากต้องการคงผ่าน reload ให้ใช้ sessionStorage และล้างเมื่อ logout/refresh ใช้ไม่ได้
  ไม่บันทึก token/password ลง console, URL, analytics หรือ Git; ไม่มี privileged key ใน FE
- ทุก endpoint ที่ต้องล็อกอินใช้ Authorization: Bearer <access_token>
- 401 จาก protected API: refresh แบบ single flight หนึ่งครั้งแล้ว retry เดิมหนึ่งครั้ง; ไม่วน refresh บน 403/503

## Endpoints

| Method / path ต่อจาก /api/v1 | Request | Success |
|---|---|---|
| POST /auth/register | {email,password,display_name} | 201 data:{user,session,email_confirmation_required} |
| POST /auth/confirm | {email,token} | 200 data:{user,session} |
| POST /auth/login | {email,password} | 200 data:{user,session} |
| POST /auth/refresh | {refresh_token} | 200 data:{user,session} |
| POST /auth/logout | Bearer, ไม่มี body | 204 ไม่มี body; scope=local ที่ server กำหนด |
| GET /me | Bearer | 200 data:{user,profile} |
| PATCH /me | Bearer, {display_name} | 200 data:{profile} |
| PATCH /me/password | Bearer, {current_password,new_password} | 200 data:{user,session}; FE แทน session เดิม |
| GET /admin/products | Bearer admin, category/search/page/page_size เดิม | 200 data:{items,page,page_size,total}; รวม inactive |
| POST /admin/products | Bearer admin, ProductWrite ด้านล่าง | 201 data:{product} |
| PATCH /admin/products/{id} | Bearer admin, sparse ProductWrite | 200 data:{product} |
| DELETE /admin/products/{id} | Bearer admin | 204; deactivate เสมอเพื่อรักษา references |

user เป็น JSON ของ Supabase ตามที่ provider คืน ไม่เพิ่ม app role ลง user_metadata
session เป็น `{access_token,refresh_token,token_type,expires_in,expires_at}` ของ Supabase
expires_at ใช้ JWT exp ถ้า provider ไม่คืน field นี้; token_type=bearer; session=null เมื่อยังต้องยืนยัน email
ไม่คืน password, privileged key หรือ provider error body
profile เป็น `{id,display_name,role,created_at,updated_at}`; role enum member/admin จาก DB ที่ server ควบคุม

ProductWrite POST ต้องมี name/category/brand/price_thb/is_active; specs/ram_model_id ตาม catalog contract เดิม
id สร้าง UUID ที่ server; PATCH เปลี่ยนเฉพาะ name/brand/price_thb/is_active/specs/ram_model_id
category เปลี่ยนหลังสร้างไม่ได้ใน milestone นี้; specs เป็น replacement ของทั้ง object
CPU/MB/RAM metadata ต้องครบตาม domain เดิม; RAM model ต้องมีอยู่ใน DB ที่ provision อย่างควบคุม
ไม่มี admin endpoint สร้าง RAM models/ยกระดับ role ใน milestone นี้
Admin อนุญาตแก้ราคา catalog; validate ยังรับเพียง IDs/counts และไม่เชื่อราคาจาก build request

## ตัวอย่าง JSON (ค่า placeholder เท่านั้น)

Register request:
```json
{"email":"student@example.test","password":"EXAMPLE-ONLY-password","display_name":"Student"}
```

Register ขณะ Confirm email เปิด:
```json
{"data":{"user":{"id":"11111111-1111-4111-8111-111111111111","email":"student@example.test"},"session":null,"email_confirmation_required":true}}
```

Confirm request:
```json
{"email":"student@example.test","token":"123456"}
```

Login/Confirm/Refresh/เปลี่ยน password response:
```json
{"data":{"user":{"id":"11111111-1111-4111-8111-111111111111","email":"student@example.test"},"session":{"access_token":"ACCESS_TOKEN_PLACEHOLDER","refresh_token":"REFRESH_TOKEN_PLACEHOLDER","token_type":"bearer","expires_in":3600,"expires_at":1900000000}}}
```

GET /me:
```json
{"data":{"user":{"id":"11111111-1111-4111-8111-111111111111","email":"student@example.test"},"profile":{"id":"11111111-1111-4111-8111-111111111111","display_name":"Student","role":"member","created_at":"2026-11-11T00:00:00Z","updated_at":"2026-11-11T00:00:00Z"}}}
```

PATCH /me: `{"display_name":"New name"}`; PATCH /me/password:
```json
{"current_password":"EXAMPLE-ONLY-old","new_password":"EXAMPLE-ONLY-new"}
```

Password flow: ตรวจ current_password ด้วย password grant ของ email จาก verified user
เปรียบเทียบ user UUID ให้ตรงกัน แล้ว update password ด้วย session ที่ reauthenticate ใหม่
คืน session ใหม่นั้นให้ FE แทนของเดิม; Supabase เป็นผู้ตัดสิน password policy/recent-auth requirements
หาก project บังคับ nonce/MFA เพิ่ม ต้องตอบ error ที่ชัดเจน ไม่ข้าม provider policy
บัญชีที่มี verified MFA factor ต้องใช้ token aal2 สำหรับ protected endpoints; aal1 ตอบ 403 REAUTHENTICATION_REQUIRED
ยังไม่มี MFA challenge/UI flow ใน milestone นี้ จึงใช้บัญชีทดสอบ email/password ที่ไม่เปิด MFA ก่อน
หาก update ล้มเหลว Go พยายาม logout session ชั่วคราวที่ reauthenticate; ไม่อ้างว่าเปลี่ยนรหัสผ่านเพิกถอนทุกอุปกรณ์

## Errors / HTTP

คง `{"error":{"code":"...","message":"..."}}` ทุก route

| HTTP | code | กรณี |
|---|---|---|
| 400 | VALIDATION_ERROR | JSON/UUID/counts/fields/length ผิด รวม role/id/email ที่ส่งมาแก้ profile |
| 400 | AUTH_REQUEST_REJECTED | provider ปฏิเสธ signup/OTP/password policy; ไม่ส่ง raw error body |
| 401 | UNAUTHORIZED | ไม่มี token, token ผิด/หมดอายุ/issuer/audience ไม่ตรง |
| 401 | INVALID_CREDENTIALS | login/current_password ผิด |
| 401 | INVALID_REFRESH_TOKEN | refresh ใช้ไม่ได้/หมดอายุ |
| 403 | EMAIL_NOT_CONFIRMED | login ก่อนยืนยัน email |
| 403 | FORBIDDEN | ยืนยันตัวตนได้แต่ไม่มี admin role |
| 403 | REAUTHENTICATION_REQUIRED | provider ต้องการ nonce/MFA ที่ flow นี้ยังไม่รองรับ |
| 404 | NOT_FOUND | product/route ไม่พบ; /me ไม่มี parameter ให้เลือกเจ้าของคนอื่น |
| 409 | CONFLICT | product reference/constraint conflict |
| 413 / 415 | PAYLOAD_TOO_LARGE / UNSUPPORTED_MEDIA_TYPE | ขนาดหรือ content type ผิด |
| 429 | RATE_LIMITED | Supabase ปฏิเสธจาก rate limit |
| 503 | SERVICE_UNAVAILABLE | configuration/provider/DB ใช้ไม่ได้; ไม่ลดสิทธิ์เป็น guest หรือใช้ mock แทน |
| 500 | INTERNAL_ERROR | error ภายในที่ไม่คาดคิด |

Auth/member bodies จำกัด 64 KiB; catalog/build contract เดิมคงไว้
Auth/me responses ใช้ Cache-Control:no-store และไม่บันทึก request body/Authorization/tokens ใน logs
Signup duplicate อาจได้ obfuscated user ตาม Supabase เพื่อกัน email enumeration: FE แสดงข้อความทั่วไป

## Token verification / Logout / สิทธิ์

Go เรียก Auth GET /auth/v1/user เพื่อให้ Supabase ตรวจลายเซ็น token (รองรับ signing-key rotation/legacy)
แล้วตรวจ exp/nbf/iss/aud/sub และ UUID ให้ตรงกับ user ที่ verified; ไม่ใช้ decoded claims อย่างเดียว
iss มาจาก SUPABASE_JWT_ISSUER (default project URL/auth/v1), aud default authenticated ต้องตรง project
role อ่าน profiles ใน DB ทุกครั้ง ไม่เชื่อ role/user_metadata/app_metadata ที่ client ส่ง

Logout ระบุ scope=local เท่านั้น เพิกถอน refresh session ปัจจุบัน ไม่ใช่ทุกอุปกรณ์
FE ล้าง token ทั้งคู่แม้ network logout ล้มเหลว แต่ต้องแจ้งว่า server อาจยังไม่เพิกถอน
access JWT ที่ออกไปแล้วอาจใช้ได้จน exp: ห้ามอ้างว่า logout เพิกถอน JWT ทันที
GET user ออนไลน์ไม่ถูกอ้างเป็น session-revocation guarantee; ใช้ access token TTL สั้นตาม project policy
หน้า sign out ใน FE เดิมต้องเพิ่ม server logout เพื่อเพิกถอน refresh แล้วล้าง token ตาม flow นี้

Guest ใช้ catalog/validate เดิมได้; member ใช้ /me ของตนเอง; admin ใช้ protected product routes
Go ใช้ server-only key สำหรับ Data API RPC; direct anon/authenticated table/RPC access เป็น default-deny
RPC เฉพาะ service_role และ mutation ตรวจ actor role ใน DB ซ้ำ; ไม่มี public role escalation endpoint
สร้าง admin ด้วย SQL ของเจ้าของ project เท่านั้นหลังยืนยัน user UUID/email ใน Dashboard

## ค่าที่ต้องตั้งและผลต่อ tests

Backend: SUPABASE_URL, SUPABASE_PUBLISHABLE_KEY (หรือ legacy anon key), SUPABASE_SECRET_KEY
(หรือ legacy service_role key), SUPABASE_JWT_ISSUER, SUPABASE_JWT_AUDIENCE, CORS_ALLOWED_ORIGINS
ห้ามใส่ server-only key ใน REACT_APP_* หรือส่ง secret ในแชต ใช้ Render Environment/secret manager
local ใช้ shell environment หรือไฟล์ .env ที่ Git ignore แล้วโหลดเองอย่างระมัดระวัง

Confirm email เปิดต้องตั้ง template OTP และ SMTP/delivery ที่ใช้ได้; ทดสอบกับอีเมลที่เจ้าของควบคุม
unit tests ใช้ fake Auth/Data API และไม่ส่งอีเมลจริง; integration ต้องตั้ง local/test Supabase แยก
ยังไม่เปิด forgot-password/recovery/OAuth/สมาชิกบันทึก build ใน milestone นี้

## เอกสารทางการที่ตรวจ 2026-11-11

- [Supabase password/email confirmation](https://supabase.com/docs/guides/auth/passwords)
- [Auth HTTP API](https://github.com/supabase/auth#api)
- [User verification](https://supabase.com/docs/reference/javascript/auth-getuser)
- [JWT](https://supabase.com/docs/guides/auth/jwts), [session](https://supabase.com/docs/guides/auth/sessions)
- [Logout scopes และ JWT expiry](https://supabase.com/docs/reference/javascript/auth-signout)
- [API keys / service_role bypasses RLS](https://supabase.com/docs/guides/getting-started/api-keys)
- [Database permissions/RLS](https://supabase.com/docs/guides/api/securing-your-api)
- [Render Go](https://render.com/docs/deploy-go-nethttp), [bind/PORT](https://render.com/docs/web-services),
  [health](https://render.com/docs/health-checks), [Blueprint](https://render.com/docs/blueprint-spec)

Render จะเตรียมไฟล์/คู่มือก่อน: ไม่สร้างบริการ/ค่าใช้จ่ายหรืออ้างว่า deploy สำเร็จจนตรวจจริง
