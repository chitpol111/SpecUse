-- Explicit opt-in only. Never called by API startup or migrations.
-- psql session: SELECT set_config('app.allow_development_seed','on',false);
-- then: \i seeds/development.sql
DO $$ BEGIN
 IF current_setting('app.allow_development_seed',true) IS DISTINCT FROM 'on' THEN
  RAISE EXCEPTION 'Development seed requires explicit opt-in';
 END IF;
END $$;
-- A database demo record, not a real SKU/price or hardware-evidence claim.
INSERT INTO public.products(id,name,category,brand,price_thb,is_active)
VALUES('95000000-0000-4000-8000-000000000001','[DEVELOPMENT DEMO ONLY] Case placeholder','case','DEMO ONLY',100,true)
ON CONFLICT(id) DO NOTHING;
