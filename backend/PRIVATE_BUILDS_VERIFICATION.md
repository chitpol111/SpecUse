# Private draft verification — 2026-11-12

Branch `feat/backend-member-builds`, worktree แยกจาก `a303f7c4dead2e8ba2f0936f9f528ce83ab70dbd`
ไม่แก้ Frontend หรือ branch ที่ deploy ไม่ merge/เปลี่ยน deployment source และไม่ apply 007 กับ specuse-dev
ส่งตรวจเป็น Draft PR จาก feat/backend-member-builds เข้า feat/backend-render-ready ต่อจาก PR #11

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
ไม่ได้ทดสอบ saved-build flow กับ specuse-dev/Render จริง และยังไม่ได้ทดสอบ private draft ร่วมกับ FE ล่าสุด
ผล CI ต้องตรวจจาก SHA/Checks ของ Draft PR แยกจาก local tests; CI ของงาน Auth เดิมไม่ใช้แทนผล private draft

Backend เก็บและคืนรายการทุกแถว/จำนวนตาม request โดยไม่บังคับหมวดละหนึ่ง
metadata PATCH ไม่แก้ arrays ที่ omitted; replacement array ต้องมาจาก draft ฉบับเต็ม
BE ไม่สามารถทราบเจตนาว่าแถวที่ client ละออกเป็นการลบจริงหรือข้อจำกัด UI ต้องใช้ FE safeguard ตาม PRIVATE_BUILDS_API.md
availability เป็น catalog readiness ไม่ใช่ราคา 0/compatible/public permission; RAM ที่ owned ครบไม่บล็อกเมื่อไม่มีแพ็ก active

ก่อนเปิดใช้ต้องตกลง PR/base, ตรวจ schema/history ของฐานจริง และประสาน migration 007/deployment แยกกัน
