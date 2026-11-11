# SpecUse
เว็บช่วยจัดสเปกคอมพิวเตอร์ให้เหมาะกับลักษณะการใช้งานและงบประมาณ

## ทดลองหน้าเช็กสเปกกับ Backend

หน้า `/build` โหลดสินค้าและส่งรายการให้ Backend ตรวจความเข้ากันได้และยอดซื้อเพิ่ม
ข้อมูลที่ใช้ตอนนี้เป็น local fixture สำหรับพัฒนาเท่านั้น ไม่ได้บันทึก Build
หรือเชื่อม Supabase

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
