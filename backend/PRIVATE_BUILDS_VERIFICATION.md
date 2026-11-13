# Private draft verification — 2026-11-12

Branch `feat/backend-member-builds`, worktree แยกจาก `a303f7c4dead2e8ba2f0936f9f528ce83ab70dbd`
ไม่แก้ Frontend หรือ branch ที่ deploy ไม่ merge/เปลี่ยน deployment source และไม่ apply 007 กับ specuse-dev
ส่งตรวจเป็น Draft PR จาก feat/backend-member-builds เข้า feat/backend-render-ready ต่อจาก PR #11

ผลตรวจชุดแรกที่ `3734a95`:

| Check | ผลที่รันจริง |
|---|---|
| go test ./... -json -count=1 | 112 tests หลัก + 354 subtests ผ่าน, fail 0; 10 packages ผ่าน |
| go vet ./... | ผ่าน ไม่มี diagnostics |
| go test -race ./internal/... -count=1 | ผ่าน 9 internal packages |
| bash integration/postgres.sh | ผ่าน migrations 001–007 บนฐานว่างชั่วคราว พร้อม auth schema/roles จำลอง |
| SQL private draft | ownership รวม Admin, inactive/multiple SKU/count/order roundtrip, metadata PATCH, clear array, public rejection, RPC/table permissions, pagination, current catalog summaries/RAM packages, mid-write rollback, account anonymization ผ่าน |
| Render-style Go build | ผ่าน: go build -tags netgo -ldflags '-s -w' -o /tmp/specuse-draft-api ./cmd/api |
| Binary/curl local fixture | Catalog/legacy 11500, draft route ไม่มี token 401/ไม่มี Auth config 503, CORS/preflight ทั้ง localhost:3000 และ 127.0.0.1:3000, SIGTERM exit 0 ผ่าน |
| Docs/examples | JSON examples parse ผ่าน, API test ใช้ create-multiple.request.json จากเอกสาร, links และ bash syntax ผ่าน |
| Diff/secrets | git diff --check และ credential-pattern/manual source review ผ่าน ไม่มี actual tracked .env หรือ Frontend changes |

ราคา 11500 เป็น legacy regression fixture ไม่ใช่ราคาสินค้าจริง 7779 ที่เจ้าของรายงาน smoke test ก่อนหน้านี้
SQL integration คือ PostgreSQL container local ไม่ใช่ Supabase hosted; provider/token checks ใช้ fake Auth/httptest
ไม่ได้ทดสอบ saved-build flow กับ specuse-dev/Render จริง ผลอัตโนมัติข้างต้นไม่ใช่ผล FE integration; ผลบางส่วนที่ผู้ใช้รายงานอยู่ด้านล่าง
ผล CI ต้องตรวจจาก SHA/Checks ของ Draft PR แยกจาก local tests; CI ของงาน Auth เดิมไม่ใช้แทนผล private draft

Backend เก็บและคืนรายการทุกแถว/จำนวนตาม request โดยไม่บังคับหมวดละหนึ่ง
metadata PATCH ไม่แก้ arrays ที่ omitted; replacement array ต้องมาจาก draft ฉบับเต็ม
BE ไม่สามารถทราบเจตนาว่าแถวที่ client ละออกเป็นการลบจริงหรือข้อจำกัด UI ต้องใช้ FE safeguard ตาม PRIVATE_BUILDS_API.md
availability เป็น catalog readiness ไม่ใช่ราคา 0/compatible/public permission; RAM ที่ owned ครบไม่บล็อกเมื่อไม่มีแพ็ก active

ก่อนเปิดใช้ต้องตกลง PR/base, ตรวจ schema/history ของฐานจริง และประสาน migration 007/deployment แยกกัน

## เตรียม native Supabase E2E แยก — 2026-11-12

คู่มือ PRIVATE_BUILDS_E2E.md แยก Backend worktree ออกจาก FE ที่ยังไม่ push และใช้ Supabase local stack ใหม่
ไม่เริ่ม Supabase CLI stack/สร้าง hosted project/ใช้ credentials จริงในรอบเตรียมนี้
ตรวจ guide bash syntax/links, JSON fixtures และ Python helper syntax ผ่าน
integration/private_draft_e2e_seed.sh ผ่านบน PostgreSQL disposable พร้อม Auth schema จำลอง:
- opt-in guard และห้าม overwrite catalog เดิม
- 13 synthetic products (active11), RAM models4, ไม่สร้าง Auth users/profiles/builds หรือ hardware evidence
- export SQL catalog ให้ Go guest API บน loopback ตรวจ quotes3600/5600/2500
- inactive/RAM-unavailable404 และ CPU conflict3800/RAM conflict3850 ผ่าน
นี่ไม่ใช่ native Supabase Auth/Data API หรือ FE integration PASS; ตาราง T01–T23 ในคู่มือยังต้องบันทึกผลจริง

## ผลทดสอบร่วมบางส่วนจากผู้ใช้ และแก้ P0002 mapping — 2026-11-12

ผู้ใช้รายงานว่าทดสอบ FE กับ Backend `c9d2eba` และ Supabase local ที่มี007แล้ว:

- บันทึก/แก้ชื่อ/ลบผ่าน; metadata PATCH เก็บครบ7 items และ2 RAM models ราคา5600
- inactive และ RAM ไม่มีแพ็ก active แสดงว่าตรวจราคาไม่ได้ตามคาด
- Member B และ Admin C ไม่เห็นdraftของAในlist และข้อมูลAไม่เปลี่ยน
- แต่ GET/PATCH/DELETE draftของA และIDที่ไม่มีอยู่ตอบ503แทน404
- ยังไม่ได้ทดสอบครบ23กรณี และไม่ใช่ผล hosted Supabase/Render

สาเหตุ: PostgREST ส่ง HTTP500 พร้อม SQLSTATE `P0002` แต่ client จัดการ5xxก่อนอ่านcode
แก้เฉพาะ Data API RPC ที่มีJSON errorถูกต้องและcode `P0002` ให้เป็น `NOT_FOUND`/404ก่อน generic5xx fallback
unknown SQLSTATE, malformed response, Auth5xx, redirect และrate limitยังคงพฤติกรรมเดิม ไม่คืนรายละเอียดDB
ไม่มีการเปลี่ยนSQL/ownership/schema และไม่ต้องรันmigrationเพิ่ม

Regression tests จำลอง HTTP500/P0002 ผ่าน HTTP client จริง และ handler→service→repository:

- Member B/Admin C เข้าถึงdraftคนอื่น และA/B/Cเข้าถึงIDที่ไม่มีอยู่: GET/PATCH/DELETEต้องได้error404แบบเดียวกัน
- PATCH ที่ไม่พบ/ไม่มีสิทธิ์หยุดที่ GETตรวจเจ้าของ ไม่เรียก RPCเขียนต่อ
- ทดสอบ P0002 ของget/patch/delete RPCโดยตรง และคง503สำหรับserver errorsที่ไม่ใช่not-found

ผลรัน local หลังแก้: `go test ./... -json -count=1` ผ่าน114 testsหลัก +382 subtests =496รายการ, fail0, 10packages
`go vet ./...` ผ่าน และ `go test -race ./internal/... -count=1` ผ่าน9packages
diff/secrets review ผ่าน ไม่มีactual tracked .env หรือ Frontend changes

ผลแก้ไขนี้ยังต้องให้ผู้ใช้รันซ้ำกับ FE+native Supabase local; ไม่อ้างว่ารอบจริงหลังแก้ผ่านแล้ว
