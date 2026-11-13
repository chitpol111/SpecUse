# ผลตรวจ local — ปรับ Register ไม่ใช้ OTP, 2026-11-11

ฐาน main: `000dd9f501c035ef1f2d1f626aa5898db686dae9` (B1 merge)
งานแยก branches ต่อกัน: docs/auth-supabase-contract → feat/supabase-members → feat/backend-admin-products → feat/backend-render-ready
ไม่มี Frontend source changes, ไม่ merge/force push และไม่สร้าง paid services

| Check | ผลที่รันจริง |
|---|---|
| go test ./... -json -count=1 | 106 tests หลัก + 337 subtests ผ่าน, fail 0; 9 packages ผ่าน |
| go vet ./... | ผ่าน ไม่มี diagnostics |
| go test -race ./internal/... | ผ่านทั้ง 8 internal packages |
| bash integration/postgres.sh | ผ่าน migrations 001–006, trigger/role escalation, grants, CRUD, typed snapshot, rollback และ health RPC |
| Render build command | ผ่าน: go build -tags netgo -ldflags '-s -w' -o /tmp/specuse-render-api ./cmd/api |
| Binary HTTP smoke | health memory 200, active list/detail, legacy compatible=true/11500, POST confirm 410, CORS ทั้ง localhost:3000 และ 127.0.0.1:3000, PATCH preflight, /me ไม่มี token 401, Auth ไม่มี config 503 |
| Process lifecycle smoke | SIGTERM exit 0; production memory ถูกปฏิเสธ |
| Render Blueprint | ไฟล์ไม่เปลี่ยนจากรอบตรวจ schema ก่อนหน้า: backend/render.yaml ผ่าน JSON schema ทางการ https://render.com/schema/render.yaml.json |
| JSON examples/fixtures | JSON 65 ไฟล์ parse ผ่าน |
| Diff/secrets | git diff --check ผ่าน; ไม่พบ tracked .env จริงหรือ credential patterns; .env.example เป็น placeholders |

SQL integration ใช้ disposable postgres:17-alpine และ auth.users/roles จำลอง ไม่ใช่ hosted Supabase
ผลในตารางเป็นผลวันที่ 2026-11-11 ก่อนเพิ่ม migration 008; ตรวจผล CI ของ commit ปัจจุบันสำหรับ 008 แยก
Auth tests ใช้ stubs/httptest ไม่ใช่หลักฐานว่า Supabase ออก/ตรวจ JWT หรือส่งอีเมลจริงสำเร็จ
Binary smoke ใช้ local fixture ไม่ใช่ live database; DB demo seed ไม่มี hardware evidence จำลอง
Secret scan เป็นการตรวจ patterns และ review diff ไม่ใช่การรับประกันจากบริการสแกนภายนอก

ยังไม่มี Supabase project values หรือ Render account/URL สำหรับทดสอบ hosted services
Confirm email ต้องปิด; signup flow นี้ไม่ต้องใช้ OTP/SMTP
ต้องทำ checklist ใน SUPABASE_SETUP.md และ RENDER.md ก่อนอ้างว่าเชื่อม Supabase หรือ deploy ผ่าน
GitHub CI ต้องแยกดูตาม commit/PR; local PASS ไม่ถูกนับเป็น CI PASS

## Flow สมาชิกที่รันแบบจำลอง

- Register ได้ session จริงจาก fake provider และ email_confirmation_required=false → GET /me ได้ทันที role=member
- Login → Refresh หมุน token pair → เปลี่ยน password โดยตรวจ current_password → Logout → refresh session นั้นใช้ไม่ได้
- ไม่อ้างว่า logout ทำให้ access JWT หมดอายุทันที; test ยืนยันว่า JWT เก่าอาจยังใช้ได้
- signup ไม่มี session ตอบ 503 AUTH_CONFIGURATION_ERROR ไม่มี auto-login fallback; session ไม่ครบตอบ 503
- POST confirm เดิม 410 โดยไม่ติดต่อ provider แม้ไม่มี Supabase config; method อื่น 405
- ส่ง role/user_metadata/app_metadata ใน signup หรือ role ใน Profile ถูกปฏิเสธ; Member ใช้ session หลังสมัครเข้า Admin ทุก method ได้ 403; ไม่มี token 401
- owner provisioning ยังเป็น SQL ฝั่งเจ้าของ ไม่มี public endpoint สมัคร Admin และไม่ได้สร้างบัญชีจริงในการทดสอบ

PR #9 local: 96 tests หลัก + 328 subtests ผ่าน; PR #10 local: 101 tests หลัก + 328 subtests ผ่าน
PR #11 ผลรวมตามตารางข้างต้น ทุกช่วง fail 0; PR #8 ตรวจ contract JSON/diff (เอกสารเท่านั้น)
ผล CI ต้องตรวจแยกตาม SHA ของแต่ละ PR ไม่ใช้ผล local แทน CI

PR #11 จัดตำแหน่ง helper fields/tests และ health note ในคู่มือใหม่เพื่อเลี่ยงข้อความชนกับ parent updates
คง test functions/logic เดิมครบ เพิ่ม commit ปกติ ไม่มี merge/rebase/force push
