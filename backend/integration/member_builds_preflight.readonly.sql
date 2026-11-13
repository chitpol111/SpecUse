-- Inspection only. This file does not install migrations or call app RPCs.
-- Run as the project owner against the explicitly selected database.
-- Confirm the project reference in the Dashboard/connection settings first:
-- current_database() is usually postgres and does not identify a Supabase project.
BEGIN TRANSACTION READ ONLY;

SELECT current_database() AS database_name, current_user AS inspecting_role,
       current_setting('transaction_read_only') AS read_only,
       to_regclass('supabase_migrations.schema_migrations') AS cli_history_table;

WITH expected(name, phase) AS (VALUES
 ('products','001-006'),('product_cpus','001-006'),
 ('product_motherboards','001-006'),('product_rams','001-006'),
 ('ram_models','001-006'),('profiles','001-006'),
 ('builds','007'),('build_items','007'),('build_ram_items','007')
)
SELECT name, phase, to_regclass('public.' || name) IS NOT NULL AS present
FROM expected ORDER BY phase, name;

-- Signatures alone do not prove matching definitions. Review schema/history too.
WITH expected(signature, phase) AS (VALUES
 ('public.backend_new_user_profile()','001-006'),
 ('public.backend_profile_get(uuid)','001-006'),
 ('public.backend_profile_update(uuid,text)','001-006'),
 ('public.backend_product_json(uuid)','001-006'),
 ('public.backend_catalog_snapshot()','001-006'),
 ('public.backend_product_save(uuid,jsonb,boolean)','001-006'),
 ('public.backend_product_deactivate(uuid,uuid)','001-006'),
 ('public.backend_health()','001-006'),
 ('public.backend_member_build_private()','007'),
 ('public.backend_member_actor(uuid)','007'),
 ('public.backend_member_build_check(jsonb,boolean)','007'),
 ('public.backend_member_build_apply(uuid,jsonb)','007'),
 ('public.backend_member_build_json(uuid)','007'),
 ('public.backend_member_build_get(uuid,uuid)','007'),
 ('public.backend_member_build_list(uuid,bigint,bigint)','007'),
 ('public.backend_member_build_create(uuid,jsonb)','007'),
 ('public.backend_member_build_patch(uuid,uuid,jsonb)','007'),
 ('public.backend_member_build_delete(uuid,uuid)','007'),
 ('public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)','008')
)
SELECT e.signature, e.phase, p.oid IS NOT NULL AS present,
       p.prosecdef AS security_definer, p.proconfig AS function_settings
FROM expected e LEFT JOIN pg_proc p ON p.oid=to_regprocedure(e.signature)
ORDER BY e.phase,e.signature;

SELECT table_name,column_name,data_type,is_nullable,column_default
FROM information_schema.columns
WHERE table_schema='public' AND table_name IN
 ('products','product_cpus','product_motherboards','product_rams',
  'ram_models','profiles','builds','build_items','build_ram_items')
ORDER BY table_name,ordinal_position;

SELECT c.relname AS table_name,c.relrowsecurity AS rls_enabled,
       con.conname,pg_get_constraintdef(con.oid) AS constraint_definition
FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
LEFT JOIN pg_constraint con ON con.conrelid=c.oid
WHERE n.nspname='public' AND c.relname IN
 ('products','product_cpus','product_motherboards','product_rams',
  'ram_models','profiles','builds','build_items','build_ram_items')
ORDER BY c.relname,con.conname;

SELECT n.nspname AS schema_name,c.relname AS table_name,
       t.tgname,t.tgenabled,pg_get_triggerdef(t.oid) AS trigger_definition
FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid
JOIN pg_namespace n ON n.oid=c.relnamespace
WHERE NOT t.tgisinternal AND
 ((n.nspname='auth' AND c.relname='users') OR
  (n.nspname='public' AND c.relname='builds'))
ORDER BY n.nspname,c.relname,t.tgname;

SELECT tablename,indexname,indexdef FROM pg_indexes
WHERE schemaname='public' AND tablename IN ('builds','build_items','build_ram_items')
ORDER BY tablename,indexname;

SELECT tablename,policyname,roles,cmd FROM pg_policies
WHERE schemaname='public' AND tablename IN ('builds','build_items','build_ram_items')
ORDER BY tablename,policyname;

-- After 007: all three tables must have RLS and no direct grants for these roles.
WITH tables(name) AS (VALUES ('builds'),('build_items'),('build_ram_items')),
roles(name) AS (VALUES ('anon'),('authenticated'),('service_role'))
SELECT t.name AS table_name,r.name AS role_name,
       c.oid IS NOT NULL AS table_present,pr.oid IS NOT NULL AS role_present,
       CASE WHEN c.oid IS NOT NULL AND pr.oid IS NOT NULL THEN
        has_table_privilege(pr.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
       END AS has_any_direct_table_privilege
FROM tables t CROSS JOIN roles r
LEFT JOIN pg_class c ON c.oid=to_regclass('public.' || t.name)
LEFT JOIN pg_roles pr ON pr.rolname=r.name
ORDER BY t.name,r.name;

-- After 007/008: CRUD and checked-save RPC EXECUTE is service_role only; helpers are not exposed.
WITH expected(signature, kind) AS (VALUES
 ('public.backend_member_build_private()','helper'),
 ('public.backend_member_actor(uuid)','helper'),
 ('public.backend_member_build_check(jsonb,boolean)','helper'),
 ('public.backend_member_build_apply(uuid,jsonb)','helper'),
 ('public.backend_member_build_json(uuid)','helper'),
 ('public.backend_member_build_get(uuid,uuid)','rpc'),
 ('public.backend_member_build_list(uuid,bigint,bigint)','rpc'),
 ('public.backend_member_build_create(uuid,jsonb)','rpc'),
 ('public.backend_member_build_patch(uuid,uuid,jsonb)','rpc'),
 ('public.backend_member_build_delete(uuid,uuid)','rpc'),
 ('public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)','rpc')
), roles(name) AS (VALUES ('anon'),('authenticated'),('service_role'))
SELECT e.signature,e.kind,r.name AS role_name,p.oid IS NOT NULL AS present,
       CASE WHEN p.oid IS NOT NULL AND pr.oid IS NOT NULL THEN
        has_function_privilege(pr.oid,p.oid,'EXECUTE') END AS can_execute,
       CASE WHEN p.oid IS NOT NULL THEN EXISTS(
        SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE a.grantee=0 AND a.privilege_type='EXECUTE') END AS public_can_execute
FROM expected e CROSS JOIN roles r
LEFT JOIN pg_proc p ON p.oid=to_regprocedure(e.signature)
LEFT JOIN pg_roles pr ON pr.rolname=r.name
ORDER BY e.kind,e.signature,r.name;

ROLLBACK;
