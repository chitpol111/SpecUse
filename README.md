# SpecUse
เว็บช่วยจัดสเปกคอมพิวเตอร์ให้เหมาะกับลักษณะการใช้งานและงบประมาณ

## ทดลองหน้าเช็กสเปกกับ Backend

หน้า `/build` โหลดสินค้าและส่งรายการให้ Backend ตรวจความเข้ากันได้และยอดซื้อเพิ่ม
Backend local fixture ตามคำสั่งด้านล่างใช้สำหรับพัฒนา catalog/validation เท่านั้น
ไม่บันทึก Build หรือเชื่อม Supabase
หน้า Build เลือก CPU, เมนบอร์ด, RAM, GPU, Storage, PSU และ Case ได้อย่างละหนึ่งรุ่น
โดย Backend คำนวณยอดจาก catalog และตรวจด้วยโหมด `gpu_storage_v1` ซึ่งแสดง
ผลพื้นฐานของ CPU socket และชนิด RAM ที่เลือก พร้อมแสดงข้อขัดแย้งอื่นที่ตรวจพบจริง
การตรวจยังไม่ครอบคลุมอุปกรณ์ทุกส่วนและไม่ยืนยันว่าเครื่องประกอบได้ครบ
RAM เลือกเป็นรุ่นและระบุจำนวนโมดูล ส่วนอุปกรณ์หมวดอื่นเลือกได้หนึ่งชิ้นต่อหมวด

หน้า `/login` และ `/register` เชื่อม Auth API ของ Go ตาม [Auth contract](backend/AUTH_API.md)
Backend fixture ตามคำสั่งด้านล่างยังไม่รองรับ Auth; ต้องตั้งค่าและทดสอบ hosted services จริง
หน้า `/profile` ใช้ข้อมูลจาก `/me` และรองรับแก้ชื่อที่แสดงกับเปลี่ยนรหัสผ่าน
Session เก็บในหน่วยความจำเท่านั้น จึงต้องเข้าสู่ระบบใหม่เมื่อรีเฟรชหรือปิดหน้าเว็บ
สมาชิกสามารถบันทึกสเปกส่วนตัวจากหน้าสรุป และดู แก้ชื่อ/คำอธิบาย หรือลบที่
`/me/builds` รายการอุปกรณ์ในสเปกที่บันทึกแสดงแบบอ่านอย่างเดียวเพื่อรักษาหลาย
SKU และ RAM models รวมถึงสินค้าที่ปิดใช้งาน ราคาต้องตรวจใหม่และอาจตรวจไม่ได้
ยังไม่มีการแชร์สาธารณะ
ฟีเจอร์นี้ต้องใช้ Backend private draft และฐานที่ติดตั้ง migrations ตาม
[แผน release](backend/PRIVATE_BUILDS_RELEASE_PLAN.md);
Backend ที่ deploy อยู่ยังไม่รองรับ และยังไม่ได้ทดสอบร่วมกับ hosted Supabase
หน้า `/admin/products` สำหรับบัญชี Admin รองรับค้นหา กรอง เพิ่ม แก้ไข และปิดใช้งานสินค้า
โดย Backend เป็นผู้ตรวจสิทธิ์และข้อมูลจริง; หมวดหมู่เปลี่ยนไม่ได้ และการปิดใช้งานไม่ลบข้อมูล
ฟอร์มมีช่องสเปกตามหมวดสินค้า ส่วนข้อมูลขั้นสูงยังแก้ได้ด้วย JSON; RAM ต้องใช้รหัสรุ่น
ที่เตรียมไว้ในฐานข้อมูลแล้ว และสเปกที่ไม่ทราบไม่ควรเดาค่า
ดู [ตั้งค่า Supabase](backend/SUPABASE_SETUP.md), [สิทธิ์ Admin](backend/ADMIN_API.md)
และ [เตรียม Render](backend/RENDER.md) สำหรับ Backend รุ่นใหม่

เปิด Terminal แรกจาก root ของ repository สำหรับ Backend:

```sh
cd backend
APP_ENV=local \
LOCAL_CATALOG_FILE=./testdata/catalog.local.json \
PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

เปิด Terminal อีกอันจาก root ของ repository สำหรับ Frontend:

```sh
cd frontend
npm ci
npm start
```

เปิด `http://localhost:3000/build` หาก Frontend หรือ Backend ใช้ URL อื่น
ให้ตั้ง `REACT_APP_API_BASE_URL` หรือค่า CORS ให้ตรงกับ origin ใน browser
Frontend ใช้ Node.js รุ่น 20.19+, 22.12+ หรือ 24 ขึ้นไป และ build ด้วย `npm run build`;
ผลลัพธ์อยู่ใน `frontend/build` โดยยังใช้ตัวแปร `REACT_APP_API_BASE_URL` เดิม
