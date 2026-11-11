# SpecUse
เว็บช่วยจัดสเปกคอมพิวเตอร์ให้เหมาะกับลักษณะการใช้งานและงบประมาณ

## ทดลองหน้าเช็กสเปกกับ Backend

หน้า `/build` โหลดสินค้าและส่งรายการให้ Backend ตรวจความเข้ากันได้และยอดซื้อเพิ่ม
ตัวอย่างคำสั่งด้านล่างใช้ local fixture สำหรับพัฒนา ไม่บันทึก Build หรือเชื่อม Supabase
Backend เพิ่มโหมด Supabase/Auth/Profile/Admin แล้ว ดู [Auth contract](backend/AUTH_API.md),
[ตั้งค่า Supabase](backend/SUPABASE_SETUP.md), [สิทธิ์ Admin](backend/ADMIN_API.md) และ
[เตรียม Render](backend/RENDER.md); ยังต้องตั้งค่าบัญชีและทดสอบ hosted services จริง

เปิด Terminal แรกสำหรับ Backend (ใช้ clone แยก `SpecUse-backend`):

```sh
cd /workspaces/SpecUse-backend/backend
APP_ENV=local \
LOCAL_CATALOG_FILE=./testdata/catalog.local.json \
PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

เปิด Terminal อีกอันสำหรับ Frontend:

```sh
cd /workspaces/SpecUse/frontend
npm start
```

เปิด `http://localhost:3000/build` หาก Frontend หรือ Backend ใช้ URL อื่น
ให้ตั้ง `REACT_APP_API_BASE_URL` หรือค่า CORS ให้ตรงกับ origin ใน browser
