# รัน local และเตรียม Go บน Render

สถานะ: มี code/config และ local tests แล้ว **ยังไม่สร้าง Render service, ยังไม่ deploy และยังไม่ทดสอบ hosted Supabase**
ไม่มี credentials จริงใน repository และไม่ได้สร้างบริการที่มีค่าใช้จ่าย
อ่าน [Auth contract](AUTH_API.md), [Supabase setup](SUPABASE_SETUP.md), [Admin](ADMIN_API.md) ก่อน

## Local: ทดลอง Catalog/Validate โดยไม่ใช้บัญชี

ต้องมี Go >=1.22 และ curl; Docker ใช้เฉพาะ integration tests
จาก root ของ clone ที่มีงานนี้ครบ (ปัจจุบัน `feat/backend-render-ready`):

```bash
cd backend
APP_ENV=local CATALOG_BACKEND=memory \
LOCAL_CATALOG_FILE=./testdata/catalog.local.json PORT=8080 \
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000 \
go run ./cmd/api
```

API base URL `http://127.0.0.1:8080/api/v1`; local bind เฉพาะ loopback

```bash
curl -i http://127.0.0.1:8080/api/v1/health
curl -i http://127.0.0.1:8080/api/v1/products
curl -i -X POST http://127.0.0.1:8080/api/v1/builds/validate \
  -H 'Content-Type: application/json' --data-binary @examples/validate.request.json
```

Legacy example ต้องได้ `compatible:true,total_price_thb:11500`
Health ใน memory mode: `{"data":{"status":"ok","catalog_backend":"memory"}}`
Auth ที่ต้องติดต่อ provider ตอบ 503 ใน memory mode; protected endpoint ที่ไม่มี Bearer ตอบ 401
memory mode ไม่จำลอง login และไม่เชื่อม database

## Local: ทดสอบ Supabase ที่เจ้าของตั้งค่าแล้ว

รัน migrations 001–006 ตาม [SUPABASE_SETUP.md](SUPABASE_SETUP.md) ไม่รันซ้ำโดยไม่ตรวจประวัติ
เปิด email confirmation, ตั้ง signup email template เป็น OTP และ SMTP ที่ส่งได้จริง
ตั้ง `CATALOG_BACKEND=supabase`, URL/keys/issuer/audience ใน backend environment
เอา `LOCAL_CATALOG_FILE` ออก; โปรแกรมไม่อ่าน `.env` อัตโนมัติและไม่ fallback ไป fixture
React ไม่เรียก Supabase SDK หรือเก็บ server key; ใช้ Go อย่างเดียวตาม AUTH_API.md

ตัวอย่าง request files อยู่ [examples/auth](examples/auth/README.md) เป็น placeholder เท่านั้น
ให้สร้างสำเนานอก repository สำหรับ email/password/token จริงและไม่บันทึกลง logs

```bash
export API_BASE=http://127.0.0.1:8080/api/v1
curl -i -X POST "$API_BASE/auth/register" -H 'Content-Type: application/json' \
  --data-binary @examples/auth/register.request.json
curl -i -X POST "$API_BASE/auth/confirm" -H 'Content-Type: application/json' \
  --data-binary @examples/auth/confirm.request.json
curl -i -X POST "$API_BASE/auth/login" -H 'Content-Type: application/json' \
  --data-binary @examples/auth/login.request.json
# ACCESS_TOKEN ต้องตั้งผ่านช่องทางเก็บ secret ของเครื่อง ไม่พิมพ์ค่าในคู่มือหรือ source
curl -i "$API_BASE/me" -H "Authorization: Bearer $ACCESS_TOKEN"
curl -i -X PATCH "$API_BASE/me" -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' --data-binary @examples/auth/profile.request.json
curl -i -X POST "$API_BASE/auth/refresh" -H 'Content-Type: application/json' \
  --data-binary @examples/auth/refresh.request.json
curl -i -X PATCH "$API_BASE/me/password" -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' --data-binary @examples/auth/password.request.json
curl -i -X POST "$API_BASE/auth/logout" -H "Authorization: Bearer $ACCESS_TOKEN"
```

หลัง refresh/password ให้แทน token pair ทั้งชุดก่อนใช้ต่อ; logout ต้องล้างทั้งคู่
Access JWT เดิมอาจใช้ได้จนหมดอายุ แม้ refresh session ถูกเพิกถอนแล้ว
ข้อมูล DB demo ไม่ใช่ fixture ยอด 11500 และไม่มี synthetic hardware evidence ใน production

## Render: สิ่งที่เจ้าของบัญชีต้องทำเอง

1. เชื่อม repository กับ Render และเลือก branch ที่ผ่าน review; ก่อน merge งานนี้ต้องใช้ branch ที่มีโค้ดครบ
   หลัง merge แต่ละช่วงจึงค่อยเปลี่ยนเป็น main โดยตรวจ commit ที่ deploy
2. ใช้ Web Service runtime Go ตั้ง Root Directory `backend`
   หรือใช้ Blueprint path `backend/render.yaml` (ไฟล์อยู่ใต้ backend ต้องเลือก path ให้ถูก)
3. ตรวจ plan และราคาใน Dashboard ก่อนกดยืนยัน; YAML ระบุ `plan: free` และปิด automatic deploy
   หาก Free ใช้ไม่ได้ให้หยุดเพื่อให้เจ้าของเลือกเอง ห้ามเปลี่ยนเป็น paid โดยอัตโนมัติ
   การกดสร้าง Blueprint อาจเริ่ม deployment แรก ต้องพร้อมค่าตั้งก่อน
4. Build: `go build -tags netgo -ldflags '-s -w' -o bin/api ./cmd/api`
   Start: `./bin/api`; ต้องใช้ Go >=1.22 (ตรวจ build log จริง)
5. Health Check Path `/api/v1/health`; Render กำหนด `PORT` ให้ ไม่ hardcode port สำหรับ hosted service
   APP_ENV=production bind `0.0.0.0:$PORT`; local mode bind 127.0.0.1 จึงห้ามใช้ APP_ENV=local บน Render
6. ตั้ง environment ด้านล่างผ่าน Render Environment โดยไม่ใส่ secrets ใน Git/Frontend
   ไม่แชร์ screenshot ที่เห็น values และไม่ใส่ credentials ลง build/start command
7. Manual deploy แล้วตรวจ health/logs/flow ตาม checklist ก่อนส่ง URL ให้ Fluke
   ไม่เปิด auto-deploy จนเจ้าของตัดสินใจ; ไม่ผูก deployment กับ main ที่ยังไม่มีโค้ดชุดนี้

| Variable | ค่าที่ต้องตั้ง |
|---|---|
| APP_ENV | production |
| CATALOG_BACKEND | supabase (production memory ถูกปฏิเสธ) |
| SUPABASE_URL | project HTTPS URL |
| SUPABASE_PUBLISHABLE_KEY | project publishable key หรือ legacy anon key |
| SUPABASE_SECRET_KEY | server secret key หรือ legacy service_role JWT; server เท่านั้น |
| SUPABASE_JWT_ISSUER | issuer จริง เช่น project URL + /auth/v1 |
| SUPABASE_JWT_AUDIENCE | authenticated หรือตรงกับ project policy |
| CORS_ALLOWED_ORIGINS | exact Frontend origins คั่น comma ไม่มี trailing slash |
| PORT | ใช้ค่าที่ Render inject; local default 8080 |

ห้ามตั้ง LOCAL_CATALOG_FILE บน Render; health ใน Supabase mode เรียก restricted DB RPC ด้วย deadline 2s
200: `{"data":{"status":"ok","catalog_backend":"supabase"}}`
503: `{"error":{"code":"SERVICE_UNAVAILABLE","message":"บริการยังไม่พร้อมใช้งาน"}}`
health ไม่เปิดเผย credentials หรือ provider body และไม่ได้พิสูจน์ Auth/SMTP ว่าครบทุก flow

CORS สำหรับ development: `http://localhost:3000,http://127.0.0.1:3000`
สำหรับ hosted React ให้เพิ่ม origin Vercel จริงที่เจ้าของยืนยัน เช่น `https://YOUR-FRONTEND.vercel.app`
ไม่ใช้ wildcard และ preview URLs ต้องอนุญาตแบบเจาะจง
Frontend เปลี่ยนเพียง API base URL เป็น `https://YOUR-API.onrender.com/api/v1` ในงานของ Fluke
Go อนุญาต GET/POST/PATCH/DELETE และ Authorization/Content-Type ตาม preflight; ไม่มี cookies

## เกณฑ์ตรวจบริการจริงก่อน demo

- จด deploy commit, วันที่, URL และผลตรวจจริง; health ต้อง 200 mode=supabase
- Register → email OTP → Confirm → Login → GET/PATCH me → Refresh → Password → Logout
- ทดสอบ password เดิมใช้ไม่ได้หลังเปลี่ยน และ refresh ของ session ที่ logout ใช้ไม่ได้
  อย่าอ้างว่า access JWT ถูกเพิกถอนทันทีหรือ password เปลี่ยนแล้วทุกอุปกรณ์ถูก logout
- token ผิด/หมดอายุ 401; member เข้า Admin 403; role/id ใน profile body 400
- เจ้าของ provision Admin ตาม ADMIN_API.md; CRUD/ราคา/active ผ่าน DB แล้ว public Catalog/Validate เห็นข้อมูลตรงกัน
- Direct anon/authenticated table/RPC access ต้องถูกปฏิเสธตาม grants/RLS
- Browser จาก Frontend จริง preflight ทั้ง origins ที่ใช้, Loading/Empty/Error ให้ Fluke ตรวจ
- ไม่พบ password/token/key ใน application logs; ตรวจ log ที่ account ควบคุมโดยไม่ส่ง secret ในแชต

## ตรวจในเครื่อง/CI

```bash
cd backend
go test ./...
go vet ./...
go test -race ./internal/...
bash integration/postgres.sh
go build -tags netgo -ldflags '-s -w' -o bin/api ./cmd/api
```

Unit tests ใช้ stubs/httptest, SQL integration ใช้ disposable PostgreSQL พร้อม auth schema/roles จำลอง
ผลเหล่านี้ไม่เท่ากับ hosted Supabase/Render integration; ไม่มี paid service ถูกสร้างโดย tests
CI เพิ่ม vet/race/SQL integration และรับ PR ทุก base เพื่อรองรับการ review branches ที่ต่อกัน
Frontend source และคำสั่ง Frontend tests/build เดิมไม่ได้แก้

## เอกสารทางการที่ตรวจ

- [Go net/http บน Render](https://render.com/docs/deploy-go-nethttp)
- [Web Service / bind / PORT](https://render.com/docs/web-services)
- [Health checks](https://render.com/docs/health-checks)
- [Blueprint spec / plan / deployment / environment](https://render.com/docs/blueprint-spec)
- [Supabase sessions](https://supabase.com/docs/guides/auth/sessions)
- [Logout / JWT expiry](https://supabase.com/docs/reference/javascript/auth-signout)
