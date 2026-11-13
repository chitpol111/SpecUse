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

ติดตั้ง migrations 001–008 ตาม [SUPABASE_SETUP.md](SUPABASE_SETUP.md)
สำหรับฐานที่มีข้อมูล ตรวจ schema/history/backup ก่อน และไม่รัน migration ที่ติดตั้งแล้วซ้ำ
สำหรับฐานเดิม ให้ตรวจ 007 และ 008 แยกกันตาม [release plan](PRIVATE_BUILDS_RELEASE_PLAN.md) ที่อนุมัติแล้ว
เปิด Email provider และ Allow new users to sign up, **ปิด Confirm email** แล้ว Save ใน Auth → Sign In / Providers → Email
ไม่ต้องรอ OTP หรือตั้ง signup SMTP สำหรับ flow หลักนี้; ดูแหล่งอ้างอิงและ local config ใน SUPABASE_SETUP.md
ตั้ง `CATALOG_BACKEND=supabase`, URL/keys/issuer/audience ใน backend environment
เอา `LOCAL_CATALOG_FILE` ออก; โปรแกรมไม่อ่าน `.env` อัตโนมัติและไม่ fallback ไป fixture
React ไม่เรียก Supabase SDK หรือเก็บ server key; ใช้ Go อย่างเดียวตาม AUTH_API.md

ตัวอย่าง request files อยู่ [examples/auth](examples/auth/README.md) เป็น placeholder เท่านั้น
ให้สร้างสำเนานอก repository สำหรับ email/password/token จริงและไม่บันทึกลง logs

```bash
export API_BASE=http://127.0.0.1:8080/api/v1
# เตรียมสำเนา request ที่มีค่าจริงนอก repository ก่อน; examples/* เป็น placeholders
AUTH_TEST_DIR=$(mktemp -d)
chmod 700 "$AUTH_TEST_DIR"
cp examples/auth/register.request.json examples/auth/login.request.json examples/auth/profile.request.json examples/auth/password.request.json "$AUTH_TEST_DIR/"
# ปิด shell tracing; response มี session จึงบันทึกใน private temp directory ไม่พิมพ์ลง logs
set +x
```

แก้สำเนาใน AUTH_TEST_DIR ก่อน: email/password ของบัญชีทดสอบที่เจ้าของควบคุมต้องตรงกันใน register/login
password.request.json ใช้ current_password เดียวกับที่สมัคร และ new_password ที่ต่างออกไป
ไม่แก้ไฟล์ tracked; refresh.request.json จะสร้างจาก response ในคำสั่งด้านล่าง
จากนั้นรันทีละขั้นและตรวจ HTTP status:

```bash
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output "$AUTH_TEST_DIR/register.response.json" -X POST "$API_BASE/auth/register" -H 'Content-Type: application/json' \
  --data-binary @"$AUTH_TEST_DIR/register.request.json"
# หลัง register ใช้ session โดยไม่ต้อง login/OTP; อ่าน token เข้า variable โดยไม่ echo
ACCESS_TOKEN=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["data"]["session"]["access_token"])' "$AUTH_TEST_DIR/register.response.json")
# ACCESS_TOKEN ไม่พิมพ์ลง console/source; ตรวจ profile.role จาก private response ว่าเป็น member
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output "$AUTH_TEST_DIR/me.response.json" "$API_BASE/me" -H "Authorization: Bearer $ACCESS_TOKEN"
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output "$AUTH_TEST_DIR/profile.response.json" -X PATCH "$API_BASE/me" -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' --data-binary @"$AUTH_TEST_DIR/profile.request.json"
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output "$AUTH_TEST_DIR/login.response.json" -X POST "$API_BASE/auth/login" -H 'Content-Type: application/json' \
  --data-binary @"$AUTH_TEST_DIR/login.request.json"
# ก่อน refresh อ่าน refresh_token จาก login.response.json แล้วสร้าง refresh.request.json ใน temp directory
python3 -c 'import json,sys; s=json.load(open(sys.argv[1]))["data"]["session"]; json.dump({"refresh_token":s["refresh_token"]},open(sys.argv[2],"w"))' "$AUTH_TEST_DIR/login.response.json" "$AUTH_TEST_DIR/refresh.request.json"
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output "$AUTH_TEST_DIR/refresh.response.json" -X POST "$API_BASE/auth/refresh" -H 'Content-Type: application/json' \
  --data-binary @"$AUTH_TEST_DIR/refresh.request.json"
ACCESS_TOKEN=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["data"]["session"]["access_token"])' "$AUTH_TEST_DIR/refresh.response.json")
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output "$AUTH_TEST_DIR/password.response.json" -X PATCH "$API_BASE/me/password" -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H 'Content-Type: application/json' --data-binary @"$AUTH_TEST_DIR/password.request.json"
ACCESS_TOKEN=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["data"]["session"]["access_token"])' "$AUTH_TEST_DIR/password.response.json")
curl --silent --show-error --fail-with-body --write-out 'HTTP %{http_code}\n' --output /dev/null -X POST "$API_BASE/auth/logout" -H "Authorization: Bearer $ACCESS_TOKEN"
```

คำสั่ง Auth ชุดนี้ต้องมี Python 3 และ curl ที่รองรับ --fail-with-body; ทุกไฟล์ค่าจริงอยู่ใน AUTH_TEST_DIR
ให้ตรวจ status สำเร็จของแต่ละคำสั่งก่อนทำต่อ หาก 503 AUTH_CONFIGURATION_ERROR ให้หยุดและให้เจ้าของแก้ Confirm email
หลัง refresh/password ให้แทน token pair ทั้งชุดก่อนใช้ต่อ; logout ต้องล้างทั้งคู่ (ตัวอย่างนี้อ่าน access ใหม่ทุกครั้ง)
ล้างตัวแปร ACCESS_TOKEN และลบ private temp files เมื่อจบทดสอบ โดยระบุเฉพาะ directory ที่สร้างในงานนี้
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
7. ก่อน Manual deploy ตรวจว่าติดตั้ง 007 และ 008, reload PostgREST schema cache และตรวจสิทธิ์ RPC แล้ว
   หากยังไม่มี 008 การสร้าง/แก้สินค้า Admin จะใช้ไม่ได้; ห้ามแก้สินค้าในช่วงที่ Backend รุ่นเก่าและใหม่ทำงานปะปนกัน
   จากนั้น Manual deploy แล้วตรวจ health/logs/flow ตาม checklist ก่อนส่ง URL ให้ Fluke
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
health ไม่เปิดเผย credentials หรือ provider body และไม่ได้พิสูจน์ Auth/สมาชิกว่าครบทุก flow

CORS สำหรับ development: `http://localhost:3000,http://127.0.0.1:3000`
สำหรับ hosted React ให้เพิ่ม origin Vercel จริงที่เจ้าของยืนยัน เช่น `https://YOUR-FRONTEND.vercel.app`
ไม่ใช้ wildcard และ preview URLs ต้องอนุญาตแบบเจาะจง
Frontend เปลี่ยนเพียง API base URL เป็น `https://YOUR-API.onrender.com/api/v1` ในงานของ Fluke
Go อนุญาต GET/POST/PATCH/DELETE และ Authorization/Content-Type ตาม preflight; ไม่มี cookies

## เกณฑ์ตรวจบริการจริงก่อน demo

- จด deploy commit, วันที่, URL และผลตรวจจริง; health ต้อง 200 mode=supabase
- Register → ใช้ session เข้า GET/PATCH me ทันที → Login → Refresh → Password → Logout
  Register ต้อง 201/session ไม่เป็น null/email_confirmation_required=false, /me role=member
  POST /auth/confirm เดิมต้อง 410; หาก provider คืน user ไม่มี session ต้อง 503 AUTH_CONFIGURATION_ERROR (ไม่แสดงหน้ารอ OTP)
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

Unit tests ใช้ stubs/httptest จำลอง session lifecycle และ permission failures, SQL integration ใช้ disposable PostgreSQL พร้อม auth schema/roles จำลอง
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
