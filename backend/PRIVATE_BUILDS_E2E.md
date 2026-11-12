# ทดสอบร่วม FE + Backend PR #12 + Supabase local แยก

คู่มือนี้เป็นขั้นตอนและ expected results ไม่ใช่ผล PASS ครบทุกกรณีของ FE+native Supabase
มีผลทดสอบบางส่วนที่ผู้ใช้รายงานจาก `c9d2eba` ดู [บันทึกผล](PRIVATE_BUILDS_VERIFICATION.md)
commit ดังกล่าวยังตอบ503เมื่อ RPC ส่ง HTTP500/P0002; ใช้ HEAD ล่าสุดของ PR #12 ที่แก้ mapping แล้วและทดสอบ T15/ID ที่ไม่มีอยู่ซ้ำ
ไม่ merge/deploy/เปลี่ยน source และไม่ใช้ specuse-dev หรือฐาน hosted ที่ใช้งานอยู่
[Contract](PRIVATE_BUILDS_API.md) · [Draft PR #12](https://github.com/chitpol111/SpecUse/pull/12)

## 1. แยก Backend โดยรักษา FE ที่ยังไม่ push

ต้องมี Bash/WSL2, Go >=1.22, Node/npm รุ่นที่ FE ใช้อยู่ (CI ใช้ Node20), Docker,
Supabase CLI, psql, curl และ Python3 แนะนำรันทั้งสามส่วนบนเครื่องเดียวกันก่อน

| ส่วน | ที่รัน | URL/port เริ่มต้น |
|---|---|---|
| FE private draft | working tree เดิมของฟลุค | http://localhost:3000 |
| Go PR #12 | worktree แยก | http://127.0.0.1:8080/api/v1 |
| Supabase Auth/Data API | local stack ใหม่ นอก Git | http://127.0.0.1:54321 |
| PostgreSQL / Studio | stack เดียวกัน | 54322 / 54323 |

Browser → Go → local Supabase เท่านั้น ไม่ต้องใส่ Supabase keys ใน FE
จาก repo ของ FE จดงานค้างแล้วเพิ่ม Backend worktree ไม่ checkout/stash/reset/merge FE:

```bash
git status --short --branch
git fetch origin
git worktree add --detach ../SpecUse-backend-e2e origin/feat/backend-member-builds
cd ../SpecUse-backend-e2e/backend
export E2E_BACKEND_DIR="$PWD"
git rev-parse HEAD
```

มี worktree PR #12 ที่อัปเดตแล้วใช้อันนั้นได้ จด Backend SHA และ FE branch/สถานะ uncommitted
อย่ารัน FE จาก frontend/ ของ Backend worktree เพราะไม่ใช่งาน FE ล่าสุดที่กำลังทดสอบ

## 2. เริ่ม Supabase local ใหม่โดยไม่ link remote

เปิด terminal สำหรับ stack โดยเฉพาะ ตั้ง path จากข้อ 1:

```bash
export E2E_BACKEND_DIR=/absolute/path/SpecUse-backend-e2e/backend
export E2E_STACK_DIR=$(mktemp -d -t specuse-drafts-e2e.XXXXXX)
chmod 700 "$E2E_STACK_DIR"
cd "$E2E_STACK_DIR"
SUPABASE_ENV=development supabase init
```

แก้ config.toml ที่เพิ่งสร้างนอก Git ใน section เดิม ไม่เพิ่ม section ชื่อซ้ำ:

```toml
[auth]
enable_signup = true
site_url = "http://localhost:3000"

[auth.email]
enable_signup = true
enable_confirmations = false
```

ตรวจ ports ของ API/DB/Studio หากชน stack อื่นให้เลือก ports ว่างและใช้ค่าใหม่ทุกขั้น
ไม่หยุด/reset stack อื่น และไม่ใช้ supabase link/db push หรือคำสั่งที่ชี้ remote

```bash
cd "$E2E_STACK_DIR"
set +x
SUPABASE_ENV=development supabase start > "$E2E_STACK_DIR/start.private.log" 2>&1
SUPABASE_ENV=development supabase status -o env > "$E2E_STACK_DIR/stack.private.env"
chmod 600 "$E2E_STACK_DIR/start.private.log" "$E2E_STACK_DIR/stack.private.env"
```

อ่าน URL/keys จาก private files ด้วยตนเอง ไม่แนบไฟล์ใน Git/แชต/ผลทดสอบ
ใช้ publishable/secret keys หาก CLI แสดง หรือ ANON_KEY/SERVICE_ROLE_KEY ซึ่ง Go รองรับ
ไม่ source ไฟล์นี้ใน FE terminal; แก้ auth config หลัง start ต้อง restart stack จาก directory นี้

## 3. ติดตั้ง 001–007 เฉพาะฐาน local ว่างใหม่

host ถูก pin ที่ 127.0.0.1; ใช้ port/password ของ local stack เท่านั้น ตัวอย่าง port54322:

```bash
set +x
export E2E_DB_PORT=54322
read -r -s -p 'Local test DB password: ' PGPASSWORD
printf '\n'
export PGPASSWORD
(
  set -e
  psql -h 127.0.0.1 -p "$E2E_DB_PORT" -U postgres -d postgres -v ON_ERROR_STOP=1 <<'SQL'
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public'
   AND table_name IN ('products','product_cpus','product_motherboards','product_rams','profiles','builds','build_items','build_ram_items','ram_models')) THEN
  RAISE EXCEPTION 'Not a fresh local test database: stop; do not rerun migrations';
 END IF;
END $$;
SQL
  for migration in "$E2E_BACKEND_DIR"/migrations/*.up.sql; do
    psql -h 127.0.0.1 -p "$E2E_DB_PORT" -U postgres -d postgres -v ON_ERROR_STOP=1 -f "$migration"
  done
  psql -h 127.0.0.1 -p "$E2E_DB_PORT" -U postgres -d postgres -v ON_ERROR_STOP=1 <<'SQL'
SELECT to_regclass('public.builds'),to_regclass('public.build_items'),
       to_regclass('public.build_ram_items'),
       to_regprocedure('public.backend_member_build_create(uuid,jsonb)');
NOTIFY pgrst, 'reload schema';
SQL
)
```

ต้องไม่มี error และตาราง/RPC ไม่เป็น null ก่อนทำต่อ หากผิดพลาดให้หยุดตรวจหรือสร้าง local project ว่างใหม่
อย่ารัน loop ซ้ำบนฐานที่ติดตั้งครึ่งหนึ่ง และไม่ใช้ loop นี้กับ specuse-dev ที่มี001–006แล้ว
ห้ามใช้ integration/bootstrap.sql หรือ member_builds.sql ตั้ง native Supabase: เป็น Auth จำลอง/destructive assertions ของ container tests

## 4. เติม synthetic catalog ครั้งเดียว

seed ไม่สร้าง Auth users/profiles/roles/builds ไม่มี SKU/specs/evidence ผู้ผลิตจริง
ต้อง opt-in และ catalog ว่าง รันซ้ำหรือมีข้อมูลอยู่แล้วจะ fail โดยไม่ overwrite:

```bash
psql -h 127.0.0.1 -p "$E2E_DB_PORT" -U postgres -d postgres -v ON_ERROR_STOP=1 \
  -c "SELECT set_config('app.allow_private_draft_e2e_seed','on',false)" \
  -f "$E2E_BACKEND_DIR/seeds/private_draft_e2e.sql"
unset PGPASSWORD
```

13 products, active11, RAM models4 ทั้งหมดขึ้นต้น [SYNTHETIC E2E ONLY]
Product UUID = 10101010-0000-4000-8000-0000000000NN; RAM model UUID = 20202020-0000-4000-8000-00000000000N

| NN / model | ข้อมูลทดสอบ |
|---|---|
| 01 CPU A / 02 CPU B | TEST_SOCKET_A / TEST_SOCKET_B ราคา1000 /1200 |
| 03 motherboard | socketA, DDR4, ราคา1500 |
| 04 /05, RAM model1 | active DDR4 single400 / pair700 |
| 06 /07 storage | ราคา200 /300 สำหรับหลายSKUและowned quantities |
| 08 /09 /10 | PSU400 / Case500 / GPU600 ราคาเท่านั้น ไม่มี hardware evidence |
| 11 storage | inactive ตั้งแต่เริ่ม |
| 12, RAM model2 | DDR4 pack inactive ทั้งกลุ่ม |
| RAM model3 | DDR4 ไม่มี SKU เลย |
| 13, RAM model4 | DDR5 single450 active สำหรับ mismatch |

Expected quotes ของ fixture ไม่ใช่ราคา7779ในระบบจริง:

- e2e-baseline.request.json: CPU1000 + board1500 + RAMซื้อ3โมดูล pair+single1100 = **3600**
- e2e-multiple.request.json: baseline + storage(3−2)×200 +(2−1)×300 +PSU400+Case500+GPU600 = **5600**
  RAM model2 required6/owned6 ไม่ซื้อเพิ่ม จึงไม่บล็อกแม้ไม่มีแพ็ก active
- e2e-ram-owned.request.json: RAM model2 required4/owned4 ซื้อเฉพาะ CPU+board = **2500**

legacy compatible ตรวจ CPU/RAM เท่านั้น ไม่รับรอง GPU/PSU/Case/ports/ความครบฮาร์ดแวร์จริง
opt-in GPU/Storage อาจ unknown เพราะไม่มี hardware profiles/evidence แต่ private draft ยัง save ได้

## 5. รัน Go และ FE

Backend terminal ใช้ keys ของ local stack ผ่าน secret manager หรือ read แบบไม่ echo:

```bash
export E2E_BACKEND_DIR=/absolute/path/SpecUse-backend-e2e/backend
cd "$E2E_BACKEND_DIR"
set +x
read -r -s -p 'Local publishable/anon key: ' SUPABASE_PUBLISHABLE_KEY
printf '\n'
read -r -s -p 'Local secret/service-role key: ' SUPABASE_SECRET_KEY
printf '\n'
export SUPABASE_PUBLISHABLE_KEY SUPABASE_SECRET_KEY
export APP_ENV=local CATALOG_BACKEND=supabase PORT=8080
export SUPABASE_URL=http://127.0.0.1:54321
export SUPABASE_JWT_ISSUER="$SUPABASE_URL/auth/v1"
export SUPABASE_JWT_AUDIENCE=authenticated
export CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
unset LOCAL_CATALOG_FILE
go run ./cmd/api
```

ใช้ port ใหม่หากปรับ config และตั้ง issuer ให้ตรง local session จริง ไม่ใช้ keys/issuer ของ specuse-dev
อีก terminal ตรวจ health mode=supabase และ catalog total11:

```bash
curl -i http://127.0.0.1:8080/api/v1/health
curl -i 'http://127.0.0.1:8080/api/v1/products?page_size=100'
```

healthไม่พิสูจน์007/ownershipจนสร้างdraftได้จริง
Frontend terminal กลับไป working tree เดิมที่มี private draft ยังไม่ push:

```bash
cd /absolute/path/to/original-frontend-working-tree/frontend
REACT_APP_API_BASE_URL=http://127.0.0.1:8080/api/v1 npm start
```

ไม่ overwrite .env เดิม ไม่ย้าย session ไป localStorage เปลี่ยน base ต้อง restart npm start
ปิด runtime mock mode ถ้ามีตามวิธีของ FE ตรวจ Network ว่าไป Go local จริง ไม่ถูก mock/ไป Render
Browser เรียก Go เท่านั้น ไม่มี Supabase server key ใน bundle/requests
ถ้า dev server เดิมใช้3000/8080 ให้หยุดเฉพาะ process ของตนหรือปรับ ports/CORS ให้ตรงกัน อย่าใช้ server เก่าที่ไม่ใช่ PR #12
Codespaces: forward FE3000/Go8080 แบบ private ใช้ HTTPS forwarded Go URL ใน FE base และ exact forwarded FE origin ใน CORS
GoยังเรียกSupabaseที่loopback ไม่ต้องเปิดDB/API portsให้Browser ถ้าDockernetworkไม่พร้อมให้แก้localตามCLIguide ไม่สลับไปฐานจริง

## 6. บัญชี A/B/C ใช้ native Auth ผ่าน Go

สมัครผ่าน FE ด้วย browser profiles แยก: A=member เจ้าของ, B=member อีกคน, C=บัญชีที่จะเป็นAdmin
ใช้บัญชีทดสอบที่ควบคุมเอง Registerต้อง201/sessionพร้อมใช้/email_confirmation_required=false และ /me role=member โดยไม่OTP
เก็บรหัสผ่านใน password manager ไม่บันทึกใน repo/ผลทดสอบ
เจ้าของตรวจ C UUID/email แล้ว promote ผ่าน local Studio SQL Editor หรือ local owner psql เท่านั้น:

```sql
UPDATE public.profiles p SET role='admin',updated_at=now()
FROM auth.users u
WHERE p.id=u.id AND p.role='member'
  AND u.id='REPLACE_WITH_LOCAL_C_USER_UUID'::uuid
  AND u.email='REPLACE_WITH_LOCAL_C_EMAIL'
RETURNING p.id,p.role;
```

ต้องหนึ่งแถว Cโหลดprofileใหม่/loginใหม่ ไม่ส่งroleในRegister/Profile ไม่มีpublicAdminsignup

## 7. สร้าง draft ที่ UI สร้างเองไม่ได้ ผ่าน Go API

ไฟล์ e2e-*.request.json ใน examples/drafts ใช้หลัง seed นี้เท่านั้น
CLI login เป็น session แยกของ A เพื่อสร้าง multi/inactive/RAM-unavailable แล้วเปิดใน FE:

```bash
export E2E_BACKEND_DIR=/absolute/path/SpecUse-backend-e2e/backend
set +x
export E2E_API_BASE=http://127.0.0.1:8080/api/v1
export E2E_RESULTS=$(mktemp -d -t specuse-drafts-results.XXXXXX)
chmod 700 "$E2E_RESULTS"
python3 -c '
import getpass,json,os,sys
b={"email":input("Local A email: "),"password":getpass.getpass("Local A password: ")}
with os.fdopen(os.open(sys.argv[1],os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600),"w") as f:json.dump(b,f)
' "$E2E_RESULTS/login.request.json"
curl --silent --show-error --fail-with-body -o "$E2E_RESULTS/login.response.json" \
  -H 'Content-Type: application/json' --data-binary @"$E2E_RESULTS/login.request.json" "$E2E_API_BASE/auth/login"
ACCESS_TOKEN=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["data"]["session"]["access_token"])' "$E2E_RESULTS/login.response.json")
for example in e2e-multiple e2e-inactive e2e-ram-unavailable e2e-ram-owned; do
  curl --silent --show-error --fail-with-body -w 'HTTP %{http_code}\n' \
    -o "$E2E_RESULTS/$example.response.json" -X POST "$E2E_API_BASE/me/builds" \
    -H "Authorization: Bearer $ACCESS_TOKEN" -H 'Content-Type: application/json' \
    --data-binary @"$E2E_BACKEND_DIR/examples/drafts/$example.request.json" || break
done
```

ต้อง201ทุกไฟล์ก่อนทำต่อ หากerrorให้ดูprivate response ไม่ถือว่าสร้างแล้ว
FEของAจะเห็นdraftsโดยไม่ต้องนำCLI tokenไปใส่Browser
หากจะตรวจราคา อย่าส่งname/summary/availabilityไปvalidate ให้ projectเหลือarrays:

```bash
python3 - "$E2E_BACKEND_DIR/examples/drafts/e2e-baseline.request.json" "$E2E_RESULTS/validate.request.json" <<'PY'
import json,sys
b=json.load(open(sys.argv[1]));json.dump({"items":b["items"],"ram_items":b["ram_items"]},open(sys.argv[2],"w"))
PY
curl --silent --show-error --fail-with-body -o "$E2E_RESULTS/validate.response.json" \
  -H 'Content-Type: application/json' --data-binary @"$E2E_RESULTS/validate.request.json" "$E2E_API_BASE/builds/validate"
```

## 8. Checklist: expected ไม่ใช่ผล PASS จนกว่าจะรันจริง

บันทึกHTTP/actual UI/arraysก่อนหลัง/BackendSHA ไม่แนบkeys/password/token/HARที่ไม่ได้redact

| ID | กรณี | Expected |
|---|---|---|
| T01 | Guest Catalog/Validate และ protected draft | public200; draftทุกmethodไม่มีtoken401 |
| T02 | Register/Login/Profile A/B/C | sessionทันที ไม่มีOTP; memberจากDBไม่ใช่metadata |
| T03 | Aไม่มีdraft | list200 items:[],total:0; Empty stateไม่ใช่error |
| T04 | Saveว่าง/ไม่ครบ/มีconflict | 201private, อ่านคืนครบ ไม่บังคับcompatibleหรือครบทุกหมวด |
| T05 | baseline CPUA+board+RAM1 req4/owned1 | legacy200, ราคา3600 CPU/RAMcompatible; save/reopenได้ |
| T06 | CPU B หรือ RAM model4 DDR5 | compatible=false; privateยังsaveได้ ไม่อ้างตรวจhardwareครบ |
| T07 | multiจากAPI→FE→rename/description→save/reopen | 7items/2RAMmodels, Storage A3/owned2, B2/owned1, RAMทุกแถวคงเดิม; legacyราคา5600 |
| T08 | UIแทนหลายSKU/qty>1ไม่ได้ | แสดงครบและปิดแก้อุปกรณ์ที่ยังไม่รองรับ; metadataPATCH omit arrays ไม่ทับเหลือหมวดละหนึ่ง/qty1 |
| T09 | inactiveเดิมและdeactivateหลังเคยมีราคา | GET200เก็บrow/summary; availabilityfalse/PRODUCT_INACTIVE; ล้างราคาเก่า แสดงตรวจราคาไม่ได้ ไม่เป็น0 |
| T10 | RAM model2ต้องซื้อเพิ่มแต่แพ็กinactiveหมด | draft201/GET200, RAM_PACKAGES_UNAVAILABLE พร้อมID; ไม่ลบmodel/counts |
| T11 | RAM model3ไม่มีSKUเลย | เปลี่ยนmodelIDในAPI request; unavailableเช่นกัน ไม่สร้างSKUหรือราคาเดา |
| T12 | RAM model2 req4/owned4 | ready=trueถ้าไม่มีissueอื่น; legacyซื้อCPU+board2500 ไม่บล็อกเพราะไม่มีแพ็กขาย |
| T13 | AdminCแก้StorageA200→250 | validate multi5600→5650 คิดซื้อ1ลูกไม่ใช่3; คืนราคา200หลังcase |
| T14 | Aมี2draft, page_size=1 | pagination/total/updated_atถูก; ไม่คืนowner_id/token |
| T15 | BและAdminCใช้A buildIDตรงผ่านAPI | GET/PATCH/DELETE404; listไม่มีA; ข้อมูลAไม่เปลี่ยน |
| T16 | MemberAเข้า/admin/products | 403; guest401; Cจัดการสินค้าได้ แต่เข้าถึงdraftคนอื่นไม่ได้ |
| T17 | ส่งowner/role/price/result/readonly/null/ownedเกิน/duplicates/UUIDผิด | 400; referenceไม่มี404; ไม่มีpartial write |
| T18 | publicพร้อมเปลี่ยนชื่อ | 409PUBLIC_BUILD_NOT_ENABLED; ชื่อและarraysไม่เปลี่ยน ไม่มีpublic UI |
| T19 | Cancelลบ แล้วยืนยันลบ | cancelไม่ยิงDELETE; ยืนยัน204ไม่มีbody; GETต่อ404 ลบเฉพาะของเจ้าของ |
| T20 | Refresh browser/logout/login | memory sessionหายต้องLoginใหม่ แต่draftยังอยู่; ไม่อ้างJWTrevokeทันที |
| T21 | Refresh session/401retry | rotated pairใช้ต่อได้; auto-refreshต้องรอexpiryจริงในlocalconfig ถ้าเรียกrefreshตรงอย่างเดียวให้บันทึกว่ายังไม่ทดสอบauto401 |
| T22 | Loading/slow network/double click/หยุดGo localชั่วคราว | Loading, กันdouble submit; Errorไม่ลบform/arraysหรือแสดงราคา0; restartแล้วretryได้ |
| T23 | FE localhost:3000 และ127.0.0.1:3000 | PATCH/DELETE preflight+Authorization/Content-Typeผ่าน; originอื่นไม่ได้ |

T07/T08 เก็บก่อน/หลังนอกGitแล้วcompareเฉพาะwrite fields (metadataอาจเปลี่ยนตามcatalog):

```bash
python3 - "$E2E_RESULTS/before.json" "$E2E_RESULTS/after.json" <<'PY'
import json,sys
def rows(path):
 b=json.load(open(path))["data"]["build"]
 return ([{k:r[k] for k in ("product_id","quantity","owned_quantity")} for r in b["items"]],
         [{k:r[k] for k in ("ram_model_id","required_modules","owned_modules")} for r in b["ram_items"]])
assert rows(sys.argv[1])==rows(sys.argv[2]),"Draft requirements changed"
print("PASS: all requirement rows, quantities and order preserved")
PY
```

หากT07/T08ไม่ผ่านให้หยุดintegration ไม่แก้ด้วยลดจำนวน/ลบทิ้งเงียบ ๆ
replacement arraysแทนทั้งชุด BackendแยกการลบจริงจากUIละข้อมูลไม่ได้
ถ้าใช้opt-inแทนlegacyให้บันทึกmode/coverageและunknown/failedตามcontractเดิม ไม่ใช้expected compatible=trueของlegacyแทน

## 9. Debug และปิดรอบ

- health503: local URL/keys, CATALOG_BACKEND=supabase, Data API/backend_health; อย่าfallbackmemoryเพื่ออ้างjoint testผ่าน
- draft404: ตรวจว่า process ใช้ Backend PR #12 จริงและ FE base เป็น Go local ไม่ใช่ Render/main
- health200แต่draft503: ตรวจ007 functions/privilegesและschema cacheในlocalStudio
- Login200แต่/me401: ดูiss/aud/expของlocal sessionเทียบGo (decodedclaimsใช้ดูconfigเท่านั้น ไม่แทนsignature verification) แล้วrestart/loginใหม่
- Save400: serializeเฉพาะwrite fields/owned countsครบ ไม่spread GET responseทั้งก้อน
- CORS: ใช้Originจริงในbrowser ไม่ใช่APIorigin; restartFEเมื่อเปลี่ยนbase
- Direct Supabase RLS: local anon/authenticatedต้องอ่าน/เขียนตาราง/RPCที่ป้องกันไม่ได้; server keyไม่อยู่ในBrowser

จดcaseเป็น PASS/FAIL/NOT RUN พร้อมCLI/Go/Node versionsและSHAs เก็บหลักฐานredactแล้ว
หยุดFE/Goด้วยCtrl+C หยุดเฉพาะlocalstackนี้จากdirectoryเดิม:

```bash
cd "$E2E_STACK_DIR"
SUPABASE_ENV=development supabase stop
```

ไม่ใช้--all/--no-backupที่กระทบstack/dataอื่น เก็บprivate filesในที่ควบคุมและลบเองเมื่อจบทดสอบ
ยังไม่merge/deploy/apply007บนspecuse-devจนreviewผลและประสานขั้นตอนถัดไป

## ตรวจเฉพาะ fixtures โดยไม่เริ่ม native Supabase

```bash
cd "$E2E_BACKEND_DIR"
bash integration/private_draft_e2e_seed.sh
```

สคริปต์สร้าง PostgreSQL container ชั่วคราวกับ Auth schema จำลอง ตรวจ guard/seed13สินค้า4models
แล้ว export catalog ให้ Go guest API บน loopback ตรวจราคา3600/5600/2500 และข้อผิดพลาด/conflicts
ไม่ใช่ FE+native Auth E2E และไม่มี credentials จริง; container/temp files ถูกลบเมื่อจบ

## เอกสารทางการที่ตรวจ 2026-11-12

- [Local Supabase CLI](https://supabase.com/docs/guides/local-development/cli/getting-started)
- [Config / enable_confirmations](https://supabase.com/docs/guides/local-development/cli/config#auth.email.enable_confirmations)
- [CLI status/stop](https://supabase.com/docs/reference/cli/introduction)
- [Schema reload](https://supabase.com/kb/troubleshooting/refresh-postgrest-schema)
