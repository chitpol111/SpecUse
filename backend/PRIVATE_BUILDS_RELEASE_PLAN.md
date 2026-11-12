# แผนรวมงาน / migration 007 / deploy private draft

เตรียมเมื่อ 2026-11-12 เวลาไทย เป็นแผนสำหรับ review ยังไม่อนุญาตให้รันขั้น mutation
รอบนี้ไม่ได้ merge/retarget PR, เปลี่ยน deployment source, deploy หรือใช้ฐาน hosted ผ่าน SQL
ไม่แก้ implementation/Frontend; การส่งเอกสารแผนเข้า PR ไม่ใช่การอนุมัติให้ merge, migrate หรือ deploy

## สถานะที่ตรวจแล้ว

| งาน | Head ที่ตรวจ | Base ปัจจุบัน | สถานะ |
|---|---|---|---|
| main | 000dd9f | — | ยังไม่มี Auth/Admin/private draft |
| PR #8 contract | 2bafaee | main | open; Backend/Frontend CI ผ่าน |
| PR #9 Supabase/member | ff24aea | docs/auth-supabase-contract | open; CI ผ่าน |
| PR #10 Admin | 5cda628 | feat/supabase-members | open; CI ผ่าน |
| PR #11 Render | a303f7c | feat/backend-admin-products | open; CI ผ่าน |
| PR #12 private draft | 126a51e | feat/backend-render-ready | draft/open; Backend/Frontend CI ผ่าน |
| FE Auth/Admin | 9295738 | — | branch feat/frontend-auth-admin |
| FE private draft | 2610f2a | สืบต่อ FE Auth/Admin | push แล้ว; ยังไม่มี FE PR ในรายการที่ตรวจ |

ตรวจ refs ด้วย git fetch และสถานะ PR ผ่าน GitHub/gh แบบอ่านอย่างเดียว
FE 2610f2a รวม Auth/Admin/หมวดเพิ่มเติมเดิมอยู่แล้ว ไม่ต้อง cherry-pick ซ้ำ
ไม่พบ PR-triggered CI ของ2610f2a; ผล mock/local ที่ผู้ใช้รายงานไม่แทน CI ของ merge candidate
Backend PR CI ที่ผ่านยังใช้ FE เก่าบน branch Backend ไม่ใช่ FE private draft ล่าสุด

ข้อควรระวังของ ancestry: heads #8→#9, #9→#10 และ #10→#11 ไม่เป็น ancestor กันแบบเส้นเดียว
เพราะชุดแก้ no-OTP เคยทำแยกบนแต่ละ branch อย่าสมมติว่า retarget แล้วจะไม่มี overlap/conflict
README.md เป็นไฟล์ที่ BE/FE แก้ร่วมกัน และการตรวจ three-way file merge ใน temporary files พบ conflict
ไม่ได้แก้ conflict หรือสร้าง merge commit ในรอบนี้

จาก GET production แบบอ่านอย่างเดียว: Go /api/v1/health ตอบ200, catalog_backend=supabase;
GET /products?page_size=1 ตอบ200 และ total=5 ณ เวลาตรวจ ไม่ใช่หลักฐานว่า007ติดตั้งแล้ว
Render source a303f7c/Auto-Deploy On และ Vercel source FE Auth/Admin เป็นข้อมูลจากเจ้าของก่อนหน้า
ยังไม่มีสิทธิ์ Dashboard/API สำหรับยืนยันค่าปัจจุบัน ต้องตรวจอีกครั้งก่อนเริ่ม window
ไฟล์ backend/render.yaml ระบุ autoDeployTrigger=off แต่ไม่ใช่หลักฐานว่า Dashboard ปิดแล้ว

## Gate 0 — ตรวจฐานและ deployment settings ก่อนเริ่ม

เครื่องนี้ไม่มี Supabase/DB/Render credentials และไม่มี psql/Supabase CLI
จึงยังยืนยัน schema/history ของ specuse-dev ไม่ได้ ให้เจ้าของทำ read-only checks ด้านล่างก่อน
ไม่ส่ง keys/password/connection string ในแชต ตั้งผ่าน secret manager หรือ owner connection ที่ควบคุมได้

1. ใน Dashboard ระบุ project reference ของ specuse-dev เทียบ SUPABASE_URL ใน Go ที่ใช้งานจริง
   current_database() ซึ่งมักเป็น postgres ไม่พิสูจน์ว่าเลือก project ถูก
2. ใช้ SQL Editor ของ project นี้รันเฉพาะ
   [member_builds_preflight.readonly.sql](integration/member_builds_preflight.readonly.sql)
   สคริปต์ BEGIN READ ONLY → SELECT metadata → ROLLBACK ไม่เรียก RPC/อ่านข้อมูลผู้ใช้/แก้ตาราง
3. ถ้า cli_history_table ไม่เป็น null ตรวจเฉพาะ versions ใน history เพิ่ม:

```sql
-- เฉพาะเมื่อ query ก่อนหน้าพิสูจน์ว่ามีตารางนี้
SELECT version FROM supabase_migrations.schema_migrations ORDER BY version;
```

4. เทียบ schema/constraints/triggers/function definitions กับ migrations001–006 และบันทึกวิธีที่เคยติดตั้ง
   signature/existence ไม่พิสูจน์ว่า definition ตรงกัน; export schema แบบ owner ไว้นอก Gitเพื่อเทียบเมื่อจำเป็น
   ก่อนติดตั้ง007ต้องมี dependencies001–006ครบ และไม่มีตาราง/ฟังก์ชัน007อยู่เลย
   หากมี007บางส่วนหรือมีครบแล้วให้หยุดตรวจประวัติ/นิยาม ไม่รันซ้ำ ไม่เติม IF NOT EXISTS เพื่อข้ามความต่าง
5. อย่ารัน001–006ซ้ำ, seed, integration/bootstrap.sql หรือ integration/member_builds.sql บนฐานนี้
   อย่าใช้ supabase db push/reset/repair จนรู้ว่า history/ไฟล์ local ตรงกัน:
   repo นี้ใช้ backend/migrations/*.up.sql ไม่ใช่ Supabase CLI layout; CLI history อาจไม่ได้บันทึก SQL ที่รันก่อนหน้า
6. เจ้าของเตรียม backup/export ที่ restore ได้ พร้อม schema และข้อมูลเดิมที่ต้องรักษา เก็บนอก Git
   ตรวจสิทธิ์/แผนบริการจริงก่อน อย่าสมมติว่า Free tier มี Dashboard backup/PITR พร้อมใช้
7. บันทึก live Go SHA/deploy ID และ FE SHA/deployment ID, source branches, auto-deploy settings และ environment names
   ต้องเลือก successful deploy เดิมที่ artifact ยังใช้ rollback ได้ และจดขั้นตอนคืน source/environment
   ตรวจว่าไม่มี deployment กำลังรัน/queued อยู่ก่อนเปลี่ยนแผน

**Gate ผ่านเมื่อ:** เจ้าของยืนยัน project, dependencies/history,007ยังไม่ติดตั้ง, backup และ live rollback targets
ถ้า schema/history ต่างจากคาด แผน migration ยัง blocked จนเทียบเสร็จ แม้ health จะ200

## Gate 1 — รวม code โดยยังไม่ deploy

หลังเจ้าของอนุญาต window ให้ปิด Render Auto-Deploy และคง Vercel production deployment เดิม
ตรวจ Vercel production/preview triggers ด้วยเจ้าของก่อนเปลี่ยน branch/merge
ไม่ push/merge ไป feat/backend-render-ready หรือ feat/frontend-auth-admin ที่ live ใช้อยู่
ห้าม merge #12 ขณะที่ base ยังเป็น feat/backend-render-ready เพราะอาจ deployทันที

ใช้ Create a merge commit เพื่อรักษา ancestry; repo อนุญาตวิธีนี้และปัจจุบันไม่ auto-delete branches
อย่า squash/rebase/force push ในชุด stacked PR นี้ และเก็บ source branches ไว้จนปลายทางผ่านหมด

| ลำดับ | งานที่ทำเมื่ออนุญาต | เงื่อนไขก่อน merge |
|---|---|---|
| 1 | review/merge #8 → main | ยืนยัน contract no-OTP |
| 2 | retarget #9 → main | ตรวจ diff/CI ใหม่ แล้ว merge |
| 3 | retarget #10 → main | รักษา signup/member/Admin และไม่เอา OTP กลับมา; CI ใหม่ผ่าน |
| 4 | retarget #11 → main | ตรวจ Render/config/health diff; CI ใหม่ผ่าน |
| 5 | retarget #12 → main; review แล้ว Ready for review | ต้องมี P0002 fix126a51e, private-only,007 และ CI ใหม่ผ่านก่อน merge |
| 6 | ฟลุคเปิด/review FE PR feat/frontend-private-builds → main | รวม Auth/Admin/private draftในPRเดียวได้; แก้ README conflictร่วมกันแล้ว CI ผ่าน |

retarget ต้องทำทีละขั้นหลัง PR ก่อนหน้า mergeเข้าmainแล้ว ไม่ merge child เข้า parent branch
หาก retarget แล้ว conflict ให้หยุด review ใช้ commit แก้ conflict แบบปกติบน branch ที่เกี่ยวข้อง
ไม่แก้หรือ push branch liveโดยอัตโนมัติ โดยเฉพาะ #11 ให้ประสานเจ้าของก่อน
PR #3 GPU/Storage proposal อยู่นอก release นี้ ไม่ต้อง mergeตามมา

หลังรวม codeให้ตรวจ final main tree กับ Backend126a51e ว่า backend/และCI semanticsครบ
และ frontend/ตรง FE2610f2a ยกเว้นการแก้ที่ตกลงเพิ่ม READMEเก็บคำแนะนำทั้งสองฝั่ง
จด final release SHA ใหม่: SHAนี้จะไม่ใช่2610f2aหรือ126a51eหลังมี merge commits
ต้องทดสอบ merge candidate ใหม่ ห้ามใช้ CIของแต่ละbranchแทนผล integration

```bash
# ใน isolated worktree ของ merge candidate เท่านั้น
cd backend
go test ./...
go vet ./...
go test -race ./internal/...
bash integration/postgres.sh
bash integration/private_draft_e2e_seed.sh
cd ../frontend
npm ci
npm test -- --watchAll=false
npm run build
```

ทดสอบ FE2610f2a+BE126a51e/merge candidate กับ Supabase local เดิมต่อ โดยไม่ migrate/seedซ้ำ
ผู้ใช้ยืนยันแล้วว่า CRUD/metadata7items+2RAM/ราคา5600/inactive และ P0002 ownership404 ผ่าน
ยังไม่ครบ23cases: บันทึก PASS/FAIL/NOT RUN ตาม PRIVATE_BUILDS_E2E.md
ก่อนreleaseต้องยืนยันอย่างน้อย guest401/memberAdmin403, role/owner injection400,
private/public409แบบatomic, own-data/quantities preservation, unavailable-price UI และ CORSจริง
กรณีที่ยังไม่รันต้องระบุผู้รับผิดชอบและผลกระทบ ห้ามสรุปว่า checklistผ่านครบ

## Gate 2 — ติดตั้ง007ครั้งเดียว ก่อนเปิด private draft ใน Go

ทำขั้นนี้หลังอนุมัติแผนและตรวจ Gate0/1 พร้อมแล้วเท่านั้น ให้เจ้าของคนเดียวรัน
Go/FEเดิมยังให้บริการอยู่ และ deployอัตโนมัติยังปิดเพื่อคุมลำดับ

1. ตรวจ project/schema/history/backup อีกครั้งใน window
2. ใช้เฉพาะ migrations/007_member_builds.up.sql จาก release SHAที่reviewแล้ว
   checksumไฟล์ที่ตรวจรอบนี้:
   `e74c74721fb610d72574851be61baa99696fcdee186630f35ca304bebc261dc0`
3. ถ้าประวัติเดิมติดตั้ง versioned SQL ด้วย owner connection ให้ใช้เส้นทางเดียวกันกับไฟล์007นี้
   ถ้าใช้ Supabase CLI historyจริง ให้ reconcile mappingกับownerก่อน ห้ามสลับrunnerหรือrepairเอง
   ตัวอย่าง owner psql หลังตั้ง private service/passfileนอกGitแล้ว (ยังไม่รันรอบนี้):

```bash
# PGSERVICE/PGPASSFILE ตั้งไว้ใน owner environment ไม่ใส่ URL/password ลงคำสั่งหรือhistory
psql -X -v ON_ERROR_STOP=1 \
  -c "SET lock_timeout='5s'; SET statement_timeout='60s';" \
  -f backend/migrations/007_member_builds.up.sql
```

ไฟล์มี BEGIN/COMMIT ในตัว ไม่ใช้ --single-transactionซ้อน ไม่ตัด transactionออก
ถ้าtimeout/errorให้หยุดตรวจ stateและlogที่redactแล้ว ไม่ retryโดยข้ามCREATEบางส่วน
4. จด project reference,releaseSHA,checksum,เวลา,operator และผลใน change recordนอกGit
   ไม่อ้างว่ามีSupabaseCLI historyหากติดตั้งแบบSQLเอง
5. รัน preflight read-onlyซ้ำ: ต้องมี3ตาราง+10functions007ครบ, RLSเปิด, ไม่มีclient policies,
   table privilegesของanon/authenticated/service_roleเป็นfalse,
   CRUD RPCเป็นtrueเฉพาะservice_role; helpersและPUBLIC EXECUTEต้องfalse
   ตรวจ FK/count/order constraints, builds_owner_updated_idx และ trigger privateครบ
6. หลังCOMMITแล้ว reload Data API schema cache ด้วย owner:

```sql
NOTIFY pgrst, 'reload schema';
```

ไม่มี backfill/seed/แก้ราคา/แก้บัญชีใน007 ไม่ใส่ synthetic evidenceหรือ fixture IDsลงฐานจริง
007เป็น additive; rollbackแอปให้คงschema007และdraftsไว้ ไม่DROPเพื่อถอยrelease

## Gate 3 — Deploy Go ก่อน FE

ใช้ Render service เดิม ไม่สร้างบริการ/planใหม่ เปลี่ยน sourceเป็นmainหลังตกลงกับเจ้าของ
การบันทึก source/settings อาจ trigger deployment ให้ถือเป็นขั้นdeployและทำหลัง007ผ่านเท่านั้น
Manual Deploy specific final release SHA ที่CIผ่าน แล้วตรวจ live SHAจริงในDeploys
ไม่ใช้ latest แบบไม่จด SHA และไม่เปิดAuto-Deployกลับระหว่างตรวจ

ตรวจ Root Directory=backend, Build/Start/healthตามRENDER.md และenvironmentที่ใช้อยู่:
- APP_ENV=production, CATALOG_BACKEND=supabase, PORTจากRender
- SUPABASE_URL/keys/issuer/audienceตรงspecuse-devเดิม; server keyอยู่Goเท่านั้น
- ไม่มี LOCAL_CATALOG_FILE; CORS_ALLOWED_ORIGINSมี https://spec-use.vercel.app
  และ exact preview originเฉพาะที่ตกลงใช้ ห้าม wildcard

Health200ไม่พอ ต้องตรวจด้วยบัญชีทดสอบที่เจ้าของควบคุมผ่านGoที่deploy:
- Catalog/Validateเทียบข้อมูลจริงก่อนหน้า ไม่ใช้expected5600/11500ของsynthetic fixture
- Registerได้member/sessionทันที, Login/Refresh/Profile/Password/Logoutตามcontract
- Guest protected401, Member Admin403
- Aสร้าง/อ่าน/metadataPATCH/ลบdraftได้; BและAdminC GET/PATCH/DELETE draftAได้404
  IDที่ไม่มีอยู่404ด้วย และข้อมูลAไม่เปลี่ยน
- inactive/no-active-RAMและการคงqty/ownedใช้เฉพาะข้อมูลทดสอบที่อนุมัติในproject
  ไม่deactivateหรือเปลี่ยนราคาสินค้าที่ใช้งานจริงเพื่อทดสอบโดยไม่ประสาน
- public409และmetadataไม่เปลี่ยน; CORS/preflightจากFEจริงผ่าน

## Gate 4 — Deploy FE แล้วตรวจผ่าน Browser

เมื่อGoพร้อมจึงให้ฟลุคเปลี่ยน Vercel production source/releaseไปmainที่ตกลง
REACT_APP_API_BASE_URL=https://specuse.onrender.com/api/v1 ต้องอยู่ในbuildที่deployจริง
ตรวจGo originในNetworkและCORS, /me/builds deep-link, session memory behavior,
CRUD/7items+2RAM preservation, unavailable-price UI, Loading/Empty/Error
ไม่เก็บSupabase privileged credentialsในFE
จดทั้ง FEdeploymentSHAและGoSHA ถ้าmainขยับหลังจดreleaseให้reviewใหม่ก่อนเลือกcommit
หลังsmokeผ่านและเจ้าของยืนยันจึงตัดสินใจเปิดAuto-Deployใหม่ ไม่เปิดโดยอัตโนมัติ

## Rollback และการหยุดขั้นตอน

- preflight/historyผิดหรือmigrationfail: ไม่deployGoใหม่ คงแอปเดิม ตรวจtransaction stateก่อนแก้
- Goใหม่ผิด: อย่าdeployFEใหม่; Render rollbackไปsuccessful deployเดิมที่จดไว้ คง007และdataไว้
- FEผิดหลังGoผ่าน: rollbackVercelไปdeploymentเดิม Goใหม่ยังรองรับlegacy flowได้
- ตรวจenvironment/source/auto-deployหลังrollback: Render Dashboard rollbackปิดauto-deploy
  แต่APIrollbackไม่ปิดให้ และrollbackแอปไม่ย้อนฐานข้อมูล
- หลังมีdraftsแล้วห้ามDROPตาราง/restorebackupทับข้อมูลใหม่เอง ต้องวางแผนกู้ข้อมูลแยก

## เจ้าของงานและจุดรออนุมัติ

Backend owner: review/retarget/mergeBEตามลำดับ, ownerDBpreflight/backup/007, RenderและAPI smoke
ฟลุค: FE PR/README conflictร่วมกัน, FE CI/localremainingcases, VercelและBrowser smoke
ก่อน mutationรอบถัดไปต้องได้ผลread-onlyฐาน,live settings,backup/rollbackและความเห็นชอบrelease window
ไม่ต้องส่งsecretsในแชต; ส่งได้เฉพาะผลschema/historyที่redactแล้วและdeploySHAs

## ผลตรวจเอกสาร/SQL ในรอบเตรียมนี้

- SQL preflight รันผ่านทั้งก่อนและหลัง007บน PostgreSQL17 containerชั่วคราวกับ Auth schemaจำลอง
- ตรวจว่า statementsมีเฉพาะ READ ONLY transaction, metadata SELECT และ ROLLBACK;
  ไม่เรียกappRPCและfingerprintของpublic functionsไม่เปลี่ยนหลังรัน
- Bash syntax/local linksและcredential-pattern reviewผ่าน ไม่เปลี่ยนFrontend/implementation
- ไม่ได้รันGo/React testsใหม่ในรอบเอกสาร; CIในตารางด้านบนเป็นผลที่อ่านจากPRheadsเดิม
- ผลนี้ไม่ใช่การตรวจschema/permissionsของspecuse-dev ซึ่งยังรอownerread-onlypreflight

## เอกสารอ้างอิงทางการที่ตรวจรอบนี้

- [Render deploy / branch / manual SHA / auto-deploy](https://render.com/docs/deploys)
- [Render rollback และสิ่งที่ไม่ย้อนกลับ](https://render.com/docs/rollbacks)
- [Supabase migration history / team workflow](https://supabase.com/docs/guides/deployment/database-migrations)
- [Supabase backups / Free tier export](https://supabase.com/docs/guides/platform/backups)
