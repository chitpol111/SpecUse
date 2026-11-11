# Auth request examples — placeholders เท่านั้น

Contract: [AUTH_API.md](../../AUTH_API.md); วิธีรัน/curl: [RENDER.md](../../RENDER.md)
ไฟล์ชุดนี้เป็น JSON ที่ส่งได้ตาม schema แต่ email example.test รับอีเมลไม่ได้, OTP/token/password เป็นตัวอย่าง
ทดสอบ hosted flow ต้องสร้างสำเนานอก repository แล้วแทนด้วยค่าของบัญชีทดสอบที่ควบคุมเอง
อย่าแก้ไฟล์ที่ tracked ให้มี credentials จริง และอย่าใช้ curl -v หรือบันทึก responses ที่มี session ลง logs

| File | Method / path ต่อจาก /api/v1 |
|---|---|
| register.request.json | POST /auth/register |
| confirm.request.json | POST /auth/confirm |
| login.request.json | POST /auth/login |
| refresh.request.json | POST /auth/refresh |
| profile.request.json | PATCH /me + Bearer |
| password.request.json | PATCH /me/password + Bearer |
| admin-create.request.json | POST /admin/products + Admin Bearer |

GET /me และ POST /auth/logout ไม่ต้องมี request body; ใช้ access token ของ session ปัจจุบัน
หลัง refresh/password ต้องแทน token pair ทั้งชุด; logout ล้างทั้งคู่
Response examples และ errors อยู่ AUTH_API.md; ข้อมูล user จริงส่งตาม Supabase จึงอาจมี fields เพิ่ม
Admin example สร้างข้อมูล DEMO ที่ไม่มี hardware metadata/evidence ไม่ใช่สินค้าใน production
