# ชุดส่งต่อ Backend ให้ฟลุค

## รับโค้ดและติดตั้ง

รับโค้ดจาก branch `feat/backend-api` โดย clone เป็นโฟลเดอร์แยกสำหรับ Backend:

```sh
git clone --branch feat/backend-api --single-branch https://github.com/chitpol111/SpecUse.git SpecUse-backend
cd SpecUse-backend
```

ไม่ต้องเปลี่ยนหรือ merge branch งาน Frontend เพื่อทดลอง Backend จากโฟลเดอร์แยก
หากรับเป็น ZIP ให้ใช้โฟลเดอร์ `backend/` ทั้งชุด ไม่คัดลอกเฉพาะ main.go เพราะต้องมี
packages, go.mod, fixture และ examples ครบ

ต้องติดตั้ง Go >= 1.22 และตรวจ `go version` ให้ผ่าน มี curl/curl.exe สำหรับทดลองคำสั่ง HTTP
หากใช้ ZIP ไม่จำเป็นต้องมี Git เพื่อรัน Backend และไม่ต้องมี Node/npm, Docker, PostgreSQL
หรือ Supabase credentials สำหรับ Backend local นี้

อ่าน [BUILD_API.md](BUILD_API.md) เป็นหลักสำหรับ setup บน Bash/PowerShell,
request/response/status codes และคำสั่ง curl; [README.md](README.md) อธิบายโครงสร้างและ tests
[PHASE2.md](PHASE2.md) เป็น schema/ข้อจำกัดเพิ่มเติมสำหรับงานระยะถัดไป

## ขั้นตอนทดลอง

1. clone ตามข้างต้นหรือแตก ZIP ให้ครบ เปิด terminal ที่ parent ของ backend
2. ทำคำสั่งรันตาม BUILD_API.md: APP_ENV=local, LOCAL_CATALOG_FILE=./testdata/catalog.local.json,
   PORT=8080 และ CORS_ALLOWED_ORIGINS ตาม URL จริงของ React
3. API Base URL คือ `http://127.0.0.1:8080/api/v1` บนเครื่องของฟลุคเอง
4. เปิด terminal อีกหน้าใน backend แล้ว GET `/products` ต้องเห็น total=7 และ GET product UUID
   `abcdefab-0000-0000-0000-000000000001` ต้องได้ CPU A ราคา 25000
5. POST `/builds/validate` ด้วย `examples/validate.request.json` ต้องได้ HTTP 200,
   compatible=true, total_price_thb=11500
6. ลอง `validate-incompatible.request.json`: HTTP 200, compatible=false,
   มี CPU_SOCKET_MISMATCH และ RAM_TYPE_MISMATCH, ยอด 9500
7. ลอง `validate-excess.request.json`: ซื้อ RAM 3 โมดูลด้วยแพ็กคู่ 2 แพ็ก ได้ 4 โมดูล,
   excess_modules=1 และยอดรวม motherboard+RAM 11000

Frontend ยืนยันใช้ port 3000 โดยรองรับทั้ง `http://localhost:3000` และ `http://127.0.0.1:3000`
ให้ตั้ง CORS_ALLOWED_ORIGINS เป็น `http://localhost:3000,http://127.0.0.1:3000`
หาก React เปลี่ยนเป็น port 3001 ให้เปลี่ยนทั้งสอง origins เป็น 3001 แล้วเริ่ม Backend ใหม่
API ยังเป็น port 8080; PORT กับ CORS frontend port เป็นคนละการตั้งค่า

สำหรับ JSON ให้ใช้ files ใน [examples/](examples/) ทั้ง request และ response ที่ตรงชื่อกัน
ห้ามส่งราคา, compatibility result, specs หรือ role ใน request ระบบจะตอบ 400
ไม่ต้องส่ง token หรือ cookies สำหรับ APIs ในชุดนี้

## ขอบเขตที่ฟลุคต้องทราบ

- GET Catalog คง 6 fields เดิมและส่ง specs ตามประเภทสินค้า พร้อม ram_model_id ของ RAM
- จัดกลุ่ม RAM ด้วย ram_model_id จาก GET แล้วส่งใน ram_items โดยตรง ไม่ hardcode mapping และไม่ส่งราคา/specs กลับไป
- จำนวนเงินเป็นตัวเลขเงินบาท Backend คิดเฉพาะของที่ต้องซื้อเพิ่ม Frontend จัดรูปแบบการแสดงผลเอง
- compatible=true ตรวจเฉพาะ CPU socket/RAM type กับ motherboard ไม่ยืนยันความครบหรือสิทธิ์เผยแพร่ public
- endpoint ไม่บันทึก build และยังไม่มี member/Admin/Supabase integration
- local fixture เป็นข้อมูลสมมติ; APP_ENV=local รับ connection เฉพาะ 127.0.0.1 ไม่ใช่ server ให้ทุกคนเรียกข้ามเครื่อง
- ไม่ต้องรัน SQL migrations เพื่อทดลอง fixture

## รายการไฟล์เดิมของ Phase 2 ณ commit 64c7473

รายการประวัตินี้เทียบ `fda4725` ถึง `64c7473` ก่อนส่วนขยาย Catalog specs:
แก้ไข 8 ไฟล์ เพิ่มใหม่ 23 ไฟล์ รวม 31 ไฟล์ ไม่ใช่รายการ diff ของ branch รุ่นถัดมา
ขอบเขตมีเฉพาะ Backend, เอกสาร, tests, schema และ JSON ตัวอย่าง

แก้ไข:

```text
backend/.env.example
backend/README.md
backend/cmd/api/catalog.go
backend/cmd/api/catalog_test.go
backend/cmd/api/main.go
backend/internal/handler/cors.go
backend/internal/handler/cors_test.go
backend/testdata/catalog.local.json
```

เพิ่มใหม่:

```text
backend/BUILD_API.md
backend/HANDOFF.md
backend/PHASE2.md
backend/cmd/api/api_test.go
backend/examples/validate-excess.request.json
backend/examples/validate-excess.response.json
backend/examples/validate-incompatible.request.json
backend/examples/validate-incompatible.response.json
backend/examples/validate.request.json
backend/examples/validate.response.json
backend/internal/build/compatibility.go
backend/internal/build/compatibility_test.go
backend/internal/build/model.go
backend/internal/build/ram.go
backend/internal/build/ram_test.go
backend/internal/build/service.go
backend/internal/build/service_test.go
backend/internal/handler/api.go
backend/internal/handler/build.go
backend/internal/handler/build_test.go
backend/internal/repository/build_memory.go
backend/internal/repository/build_memory_test.go
backend/migrations/002_compatibility.up.sql
```

`.env.example` มีแต่ตัวอย่าง configuration ไม่มี secrets จริง; ไม่รวม `.env` จริง, credentials,
frontend, binary หรือ test reports ในรายการ commit ส่วน ZIP ส่งต่อรวม backend ทั้งหมด
รวมไฟล์เดิมที่ต้องใช้รันด้วย จึงมีจำนวนไฟล์มากกว่ารายการเปลี่ยนแปลงนี้

## ร่างข้อความส่งให้ฟลุค

> ฟลุค ชุด Backend Catalog + POST /api/v1/builds/validate อยู่บน branch feat/backend-api ให้ clone แยกสำหรับ Backend ตาม backend/HANDOFF.md ได้เลย
> อ่าน backend/BUILD_API.md สำหรับวิธีรันด้วย Go 1.22+ และ local fixture ตั้ง CORS รองรับทั้ง localhost:3000 และ 127.0.0.1:3000 แล้ว
> Base URL คือ http://127.0.0.1:8080/api/v1 ลอง POST ด้วย examples/validate.request.json ตาม curl ในคู่มือ ต้องได้ compatible=true และยอด 11500
> GET products มี specs และ RAM ram_model_id แล้ว อ่านจาก API ไปสร้าง request validate ได้เลย ไม่ต้องทำ mapping เอง
> ตัวอย่างไม่เข้ากันและแพ็ก RAM อยู่ใน examples/ พร้อม response เทียบให้แล้ว ยังไม่ใช้ Supabase/token และไม่ต้องส่งราคาจาก Frontend
