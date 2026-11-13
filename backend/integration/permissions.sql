-- Assertions against real PostgreSQL with a minimal, simulated Auth schema.
INSERT INTO auth.users(id,email,raw_user_meta_data) VALUES
('11111111-1111-4111-8111-111111111111','member@example.test','{"display_name":"Member","role":"admin"}'),
('22222222-2222-4222-8222-222222222222','admin@example.test','{"display_name":"Admin candidate"}');
DO $$ BEGIN
 IF (SELECT role FROM public.profiles WHERE id='11111111-1111-4111-8111-111111111111')<>'member' THEN RAISE EXCEPTION 'Metadata escalation';END IF;
 IF has_function_privilege('anon','public.backend_catalog_snapshot()','EXECUTE') THEN RAISE EXCEPTION 'Anon RPC';END IF;
 IF has_function_privilege('authenticated','public.backend_profile_update(uuid,text)','EXECUTE') THEN RAISE EXCEPTION 'Member direct RPC';END IF;
 IF has_table_privilege('authenticated','public.profiles','UPDATE') THEN RAISE EXCEPTION 'Direct profile role update';END IF;
 IF has_table_privilege('service_role','public.profiles','UPDATE') THEN RAISE EXCEPTION 'Unrestricted service profile update';END IF;
 BEGIN
  PERFORM public.backend_product_save('11111111-1111-4111-8111-111111111111','{}',true);
  RAISE EXCEPTION 'Member mutation accepted';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $$;
-- This owner-only statement represents the controlled admin provisioning step.
UPDATE public.profiles SET role='admin' WHERE id='22222222-2222-4222-8222-222222222222';
SELECT public.backend_profile_update('11111111-1111-4111-8111-111111111111','New member');
SET ROLE service_role;
SELECT public.backend_health();
SELECT public.backend_product_save('22222222-2222-4222-8222-222222222222',
'{"id":"33333333-3333-4333-8333-333333333333","name":"[TEST MODEL] CPU","category":"cpu","brand":"TEST ONLY","price_thb":25000,"is_active":true,"specs":{"cpu_socket":"TEST_SOCKET"}}',true);
SELECT public.backend_product_save('22222222-2222-4222-8222-222222222222',
'{"id":"44444444-4444-4444-8444-444444444444","name":"[TEST MODEL] Board","category":"motherboard","brand":"TEST ONLY","price_thb":5000,"is_active":true,"specs":{"cpu_socket":"TEST_SOCKET","ram_type":"DDR5"}}',true);
SELECT public.backend_product_save('22222222-2222-4222-8222-222222222222',
'{"id":"44444444-4444-4444-8444-444444444444","name":"[TEST MODEL] Board","category":"motherboard","brand":"TEST ONLY","price_thb":6000,"is_active":true,"specs":{"cpu_socket":"TEST_SOCKET","ram_type":"DDR5"}}',false);
SELECT public.backend_product_deactivate('22222222-2222-4222-8222-222222222222','33333333-3333-4333-8333-333333333333');
SELECT public.backend_catalog_snapshot();
RESET ROLE;
DO $$ BEGIN
 IF has_function_privilege('anon','public.backend_health()','EXECUTE') THEN RAISE EXCEPTION 'Anon health RPC';END IF;
 IF (SELECT price_thb FROM public.products WHERE id='44444444-4444-4444-8444-444444444444')<>6000 THEN RAISE EXCEPTION 'Price not saved';END IF;
 IF (SELECT is_active FROM public.products WHERE id='33333333-3333-4333-8333-333333333333') THEN RAISE EXCEPTION 'Deactivate failed';END IF;
 IF NOT EXISTS(SELECT 1 FROM public.product_cpus WHERE product_id='33333333-3333-4333-8333-333333333333' AND cpu_socket='TEST_SOCKET') THEN RAISE EXCEPTION 'Typed metadata lost';END IF;
 IF (public.backend_product_json('44444444-4444-4444-8444-444444444444')->'specs'->>'ram_type')<>'DDR5' THEN RAISE EXCEPTION 'Snapshot metadata lost';END IF;
 BEGIN
  PERFORM public.backend_product_save('22222222-2222-4222-8222-222222222222',
  '{"id":"55555555-5555-4555-8555-555555555555","name":"Bad RAM","category":"ram","brand":"TEST","price_thb":100,"is_active":true,"ram_model_id":"66666666-6666-4666-8666-666666666666","specs":{"ram_type":"DDR5","modules_per_package":1}}',true);
  RAISE EXCEPTION 'Missing RAM model accepted';
 EXCEPTION WHEN foreign_key_violation THEN NULL;
 END;
 IF EXISTS(SELECT 1 FROM public.products WHERE id='55555555-5555-4555-8555-555555555555') THEN RAISE EXCEPTION 'Mutation not atomic';END IF;
END $$;
DO $$
DECLARE original jsonb; current_product jsonb;
BEGIN
 IF has_function_privilege('anon','public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)','EXECUTE')
    OR has_function_privilege('authenticated','public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)','EXECUTE')
    OR NOT has_function_privilege('service_role','public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)','EXECUTE') THEN
  RAISE EXCEPTION 'Checked product save privileges incorrect';
 END IF;
 original := public.backend_product_json('44444444-4444-4444-8444-444444444444');
 PERFORM public.backend_product_save_checked('22222222-2222-4222-8222-222222222222',
  jsonb_set(original,'{name}','"Updated board"'),false,original);
 BEGIN
  PERFORM public.backend_product_save_checked('22222222-2222-4222-8222-222222222222',
   jsonb_set(original,'{price_thb}','7000'),false,original);
  RAISE EXCEPTION 'Stale product update accepted';
 EXCEPTION WHEN SQLSTATE 'PT409' THEN NULL;
 END;
 current_product := public.backend_product_json('44444444-4444-4444-8444-444444444444');
 IF current_product->>'name' <> 'Updated board' OR (current_product->>'price_thb')::bigint <> 6000 THEN
  RAISE EXCEPTION 'Stale update changed product';
 END IF;
 PERFORM public.backend_product_save_checked('22222222-2222-4222-8222-222222222222',
  jsonb_set(current_product,'{price_thb}','7000'),false,current_product);
 IF (public.backend_product_json('44444444-4444-4444-8444-444444444444')->>'price_thb')::bigint <> 7000 THEN
  RAISE EXCEPTION 'Reloaded product update failed';
 END IF;
END $$;
SELECT 'PASS: trigger/role escalation, grants, CRUD, typed snapshot and atomic rollback' AS integration_result;
