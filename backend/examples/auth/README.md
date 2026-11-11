# Auth request examples — placeholders เท่านั้น

Contract: [AUTH_API.md](../../AUTH_API.md); วิธีรัน/curl: [RENDER.md](../../RENDER.md)
ไฟล์ชุดนี้เป็น JSON ที่ส่งได้ตาม schema แต่ email example.test และ token/password เป็นตัวอย่าง ต้องใช้บัญชีที่เจ้าของควบคุมเอง
ทดสอบ hosted flow ต้องสร้างสำเนานอก repository แล้วแทนด้วยค่าของบัญชีทดสอบที่ควบคุมเอง
อย่าแก้ไฟล์ที่ tracked ให้มี credentials จริง และอย่าใช้ curl -v หรือบันทึก responses ที่มี session ลง logs

| File | Method / path ต่อจาก /api/v1 |
|---|---|
| register.request.json | POST /auth/register |
| login.request.json | POST /auth/login |
| refresh.request.json | POST /auth/refresh |
| profile.request.json | PATCH /me + Bearer |
| password.request.json | PATCH /me/password + Bearer |
| admin-create.request.json | POST /admin/products + Admin Bearer |

GET /me และ POST /auth/logout ไม่ต้องมี request body; ใช้ access token ของ session ปัจจุบัน
หลัง refresh/password ต้องแทน token pair ทั้งชุด; logout ล้างทั้งคู่
Register ต้อง 201 พร้อม session ไม่เป็น null และ email_confirmation_required=false; ใช้ session เรียก GET /me ได้ทันที role=member
ไม่ต้อง OTP; POST /auth/confirm เดิมตอบ 410 ENDPOINT_RETIRED และไม่มี confirm request example แล้ว
Provider คืน user แต่ไม่มี session ต้อง 503 AUTH_CONFIGURATION_ERROR ให้หยุดและแจ้งผู้ดูแล ไม่ retry รัว
Response examples และ errors อยู่ AUTH_API.md; ข้อมูล user จริงส่งตาม Supabase จึงอาจมี fields เพิ่ม
Admin example สร้างข้อมูล DEMO ที่ไม่มี hardware metadata/evidence ไม่ใช่สินค้าใน production

| Response file | HTTP / ความหมาย |
|---|---|
| register.response.json | 201 พร้อม session ใช้ต่อได้ ไม่มี OTP |
| me.response.json | 200 profile.role=member จาก DB ไม่ใช่ Supabase user.role |
| auth-configuration-error.response.json | 503 เมื่อ provider ยังต้อง confirmation/no session |
| confirm-retired.response.json | 410 endpoint confirm เลิกใช้ |

Session/token ใน response files เป็น placeholders ไม่ใช่ credentials จริง; expires_at เป็นตัวอย่าง Unix seconds
Login/Refresh/password ใช้ envelope user/session ตาม AUTH_API.md ไม่เพิ่ม email_confirmation_required
Admin ใช้บัญชีที่เจ้าของเตรียมล่วงหน้าตาม ADMIN_API.md เท่านั้น ไม่มี public endpoint สมัคร Admin
