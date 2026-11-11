# ผลตรวจ local — 2026-11-11

ฐาน main: `000dd9f501c035ef1f2d1f626aa5898db686dae9` (B1 merge)
งานแยก branches ต่อกัน: docs/auth-supabase-contract → feat/supabase-members → feat/backend-admin-products → feat/backend-render-ready
ไม่มี Frontend source changes, ไม่ merge/force push และไม่สร้าง paid services

| Check | ผลที่รันจริง |
|---|---|
| go test ./... -json -count=1 | 102 tests หลัก + 335 subtests ผ่าน, fail 0; 9 packages ผ่าน |
| go vet ./... | ผ่าน ไม่มี diagnostics |
| go test -race ./internal/... | ผ่านทั้ง 8 internal packages |
| bash integration/postgres.sh | ผ่าน migrations 001–006, trigger/role escalation, grants, CRUD, typed snapshot, rollback และ health RPC |
| Render build command | ผ่าน: go build -tags netgo -ldflags '-s -w' -o /tmp/specuse-render-api ./cmd/api |
| Binary HTTP smoke | health memory 200, active list/detail, legacy compatible=true/11500, CORS ทั้ง localhost:3000 และ 127.0.0.1:3000, PATCH preflight, /me ไม่มี token 401, Auth ไม่มี config 503 |
| Process lifecycle smoke | SIGTERM exit 0; production memory ถูกปฏิเสธ |
| Render Blueprint | backend/render.yaml ผ่าน JSON schema ทางการ https://render.com/schema/render.yaml.json |
| JSON examples/fixtures | JSON 62 ไฟล์ parse ผ่าน |
| Diff/secrets | git diff --check ผ่าน; ไม่พบ tracked .env จริงหรือ credential patterns; .env.example เป็น placeholders |

SQL integration ใช้ disposable postgres:17-alpine และ auth.users/roles จำลอง ไม่ใช่ hosted Supabase
Auth tests ใช้ stubs/httptest ไม่ใช่หลักฐานว่า Supabase ออก/ตรวจ JWT หรือส่งอีเมลจริงสำเร็จ
Binary smoke ใช้ local fixture ไม่ใช่ live database; DB demo seed ไม่มี hardware evidence จำลอง
Secret scan เป็นการตรวจ patterns และ review diff ไม่ใช่การรับประกันจากบริการสแกนภายนอก

ยังไม่มี Supabase project values/SMTP หรือ Render account/URL สำหรับทดสอบ hosted services
ต้องทำ checklist ใน SUPABASE_SETUP.md และ RENDER.md ก่อนอ้างว่าเชื่อม Supabase หรือ deploy ผ่าน
GitHub CI ต้องแยกดูตาม commit/PR; local PASS ไม่ถูกนับเป็น CI PASS
