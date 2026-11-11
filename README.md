# SpecUse
เว็บช่วยจัดสเปกคอมพิวเตอร์ให้เหมาะกับลักษณะการใช้งานและงบประมาณ

## ทดลองหน้าเช็กสเปกกับ Backend

หน้า `/build` โหลดสินค้าและส่งรายการให้ Backend ตรวจความเข้ากันได้และยอดซื้อเพิ่ม
ข้อมูลที่ใช้ตอนนี้เป็น local fixture สำหรับพัฒนาเท่านั้น ไม่ได้บันทึก Build
หรือเชื่อม Supabase

หน้า `/login` และ `/register` เชื่อม Auth API ของ Go ตาม contract
`backend/AUTH_API.md` ใน branch Backend รุ่นใหม่ ซึ่งยังไม่รวมใน main และยังต้อง
ทดสอบกับ Supabase จริง Backend fixture ตามคำสั่งด้านล่างยังไม่รองรับ Auth
หน้า `/profile` ใช้ข้อมูลจาก `/me` และรองรับแก้ชื่อที่แสดงกับเปลี่ยนรหัสผ่าน
Session เก็บในหน่วยความจำเท่านั้น จึงต้องเข้าสู่ระบบใหม่เมื่อรีเฟรชหรือปิดหน้าเว็บ
หน้า `/admin/products` สำหรับบัญชี Admin รองรับค้นหา กรอง เพิ่ม แก้ไข และปิดใช้งานสินค้า
โดย Backend เป็นผู้ตรวจสิทธิ์และข้อมูลจริง; หมวดหมู่เปลี่ยนไม่ได้ และการปิดใช้งานไม่ลบข้อมูล

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
