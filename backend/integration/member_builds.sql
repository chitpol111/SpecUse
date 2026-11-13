-- Disposable PostgreSQL only; no hardware facts/evidence or real accounts.
INSERT INTO public.products(id,name,category,brand,price_thb,is_active) VALUES
 ('77777777-7777-4777-8777-777777777771','[TEST] Storage A','storage','TEST ONLY',100,true),
 ('77777777-7777-4777-8777-777777777772','[TEST] Storage B','storage','TEST ONLY',200,true),
 ('99999999-9999-4999-8999-999999999991','[TEST] RAM retired','ram','TEST ONLY',300,false);
INSERT INTO public.ram_models(id,ram_type) VALUES
 ('88888888-8888-4888-8888-888888888881','DDR4'),
 ('88888888-8888-4888-8888-888888888882','DDR5');
INSERT INTO public.product_rams(product_id,ram_model_id,ram_type,modules_per_package)
 VALUES('99999999-9999-4999-8999-999999999991','88888888-8888-4888-8888-888888888881','DDR4',2);

-- Force a failure after the base row/items were modified, to prove rollback.
CREATE FUNCTION public.test_draft_write_failure() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN IF NEW.required_modules=999 THEN RAISE EXCEPTION 'TEST forced write failure' USING ERRCODE='23514';END IF;RETURN NEW;END $$;
CREATE TRIGGER test_draft_failure BEFORE INSERT ON public.build_ram_items
FOR EACH ROW EXECUTE FUNCTION public.test_draft_write_failure();

DO $$ BEGIN
 IF has_table_privilege('service_role','public.builds','SELECT') OR has_table_privilege('authenticated','public.build_items','INSERT') THEN RAISE EXCEPTION 'Direct draft table privilege';END IF;
 IF has_function_privilege('anon','public.backend_member_build_get(uuid,uuid)','EXECUTE') OR has_function_privilege('authenticated','public.backend_member_build_create(uuid,jsonb)','EXECUTE') THEN RAISE EXCEPTION 'Public draft RPC';END IF;
 IF has_function_privilege('service_role','public.backend_member_build_json(uuid)','EXECUTE') THEN RAISE EXCEPTION 'Owner-bypassing helper exposed';END IF;
 IF (SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN ('builds','build_items','build_ram_items') AND c.relrowsecurity)<>3 THEN RAISE EXCEPTION 'RLS missing';END IF;
END $$;
SET ROLE service_role;
DO $$
DECLARE
 actor uuid:='11111111-1111-4111-8111-111111111111';other_actor uuid:='22222222-2222-4222-8222-222222222222';
 payload jsonb:='{"name":"Draft with hidden selections","items":[
 {"product_id":"33333333-3333-4333-8333-333333333333","quantity":3,"owned_quantity":2},
 {"product_id":"77777777-7777-4777-8777-777777777772","quantity":5,"owned_quantity":4},
 {"product_id":"77777777-7777-4777-8777-777777777771","quantity":2,"owned_quantity":1},
 {"product_id":"44444444-4444-4444-8444-444444444444","quantity":2,"owned_quantity":0}],
 "ram_items":[{"ram_model_id":"88888888-8888-4888-8888-888888888881","required_modules":4,"owned_modules":1},
 {"ram_model_id":"88888888-8888-4888-8888-888888888882","required_modules":6,"owned_modules":6}]}';
 original jsonb;r jsonb;v_id uuid;temp_id uuid;before_total bigint;
BEGIN
 original:=public.backend_member_build_create(actor,payload);v_id:=(original->>'id')::uuid;
 IF original->>'visibility'<>'private' OR original->>'description'<>'' THEN RAISE EXCEPTION 'Create defaults';END IF;
 IF jsonb_array_length(original->'items')<>4 OR jsonb_array_length(original->'ram_items')<>2 THEN RAISE EXCEPTION 'Selections collapsed';END IF;
 IF original->'items'->1->>'product_id'<>'77777777-7777-4777-8777-777777777772' THEN RAISE EXCEPTION 'Array order lost';END IF;
 IF original->'items'->0->'product_summary'->>'is_active'<>'false' THEN RAISE EXCEPTION 'Inactive product lost';END IF;
 IF original->'ram_items'->0->>'active_package_count'<>'0' OR original->'ram_items'->1->>'owned_modules'<>'6' THEN RAISE EXCEPTION 'Unavailable/owned RAM lost';END IF;
 r:=public.backend_member_build_patch(actor,v_id,'{"name":"Renamed","description":"Notes"}');
 IF r->'items'<>original->'items' OR r->'ram_items'<>original->'ram_items' THEN RAISE EXCEPTION 'Metadata update dropped selections';END IF;
 original:=r;
 -- Read summaries and RAM availability from current catalog state, not a saved copy.
 PERFORM public.backend_product_deactivate(other_actor,'77777777-7777-4777-8777-777777777772');
 r:=public.backend_member_build_get(actor,v_id);
 IF r->'items'->1->'product_summary'->>'is_active'<>'false' THEN RAISE EXCEPTION 'Stale product summary';END IF;
 PERFORM public.backend_product_save(other_actor,'{"id":"77777777-7777-4777-8777-777777777772","name":"[TEST] Storage B","category":"storage","brand":"TEST ONLY","price_thb":200,"is_active":true}',false);
 PERFORM public.backend_product_save(other_actor,'{"id":"99999999-9999-4999-8999-999999999991","name":"[TEST] RAM retired","category":"ram","brand":"TEST ONLY","price_thb":300,"is_active":true,"ram_model_id":"88888888-8888-4888-8888-888888888881","specs":{"ram_type":"DDR4","modules_per_package":2}}',false);
 r:=public.backend_member_build_get(actor,v_id);
 IF r->'ram_items'->0->>'active_package_count'<>'1' THEN RAISE EXCEPTION 'Stale RAM package availability';END IF;
 PERFORM public.backend_product_deactivate(other_actor,'99999999-9999-4999-8999-999999999991');
 -- Admin is not permitted to read/edit/delete another member's build.
 BEGIN PERFORM public.backend_member_build_get(other_actor,v_id);RAISE EXCEPTION 'Foreign read accepted';EXCEPTION WHEN no_data_found THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_patch(other_actor,v_id,'{"name":"Stolen"}');RAISE EXCEPTION 'Foreign patch accepted';EXCEPTION WHEN no_data_found THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_delete(other_actor,v_id);RAISE EXCEPTION 'Foreign delete accepted';EXCEPTION WHEN no_data_found THEN NULL;END;
 IF (public.backend_member_build_list(other_actor,1,20)->>'total')::bigint<>0 THEN RAISE EXCEPTION 'Foreign list';END IF;
 before_total:=(public.backend_member_build_list(actor,1,20)->>'total')::bigint;
 BEGIN PERFORM public.backend_member_build_create(actor,'{"name":"Public","visibility":"public","items":[],"ram_items":[]}');RAISE EXCEPTION 'Public create accepted';EXCEPTION WHEN check_violation THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_patch(actor,v_id,'{"name":"Changed","visibility":"public"}');RAISE EXCEPTION 'Public patch accepted';EXCEPTION WHEN check_violation THEN NULL;END;
 IF public.backend_member_build_get(actor,v_id)->>'name'<>'Renamed' OR (public.backend_member_build_list(actor,1,20)->>'total')::bigint<>before_total THEN RAISE EXCEPTION 'Public write not atomic';END IF;
 BEGIN PERFORM public.backend_member_build_patch(actor,v_id,'{"name":"Bad reference","items":[{"product_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","quantity":1,"owned_quantity":0}]}');RAISE EXCEPTION 'Missing reference accepted';EXCEPTION WHEN no_data_found THEN NULL;END;
 IF public.backend_member_build_get(actor,v_id)->'items'<>original->'items' THEN RAISE EXCEPTION 'Reference error changed data';END IF;
 BEGIN PERFORM public.backend_member_build_patch(actor,v_id,'{"name":"Must rollback","items":[],"ram_items":[{"ram_model_id":"88888888-8888-4888-8888-888888888881","required_modules":999,"owned_modules":0}]}');RAISE EXCEPTION 'Forced failure not triggered';EXCEPTION WHEN check_violation THEN NULL;END;
 r:=public.backend_member_build_get(actor,v_id);
 IF r->>'name'<>'Renamed' OR r->'items'<>original->'items' OR r->'ram_items'<>original->'ram_items' THEN RAISE EXCEPTION 'Partial write escaped rollback';END IF;
 BEGIN PERFORM public.backend_member_build_patch(actor,v_id,'{"owner_id":"22222222-2222-4222-8222-222222222222"}');RAISE EXCEPTION 'Owner update accepted';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_create(actor,'{"name":"Price injection","total_price_thb":0,"items":[],"ram_items":[]}');RAISE EXCEPTION 'Price field accepted';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_patch(actor,v_id,'{"items":[{"product_id":"33333333-3333-4333-8333-333333333333","quantity":1,"owned_quantity":2}]}');RAISE EXCEPTION 'Bad owned accepted';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_patch(actor,v_id,'{"items":[{"product_id":"99999999-9999-4999-8999-999999999991","quantity":1,"owned_quantity":0}]}');RAISE EXCEPTION 'RAM SKU accepted in items';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 BEGIN PERFORM public.backend_member_build_list(actor,0,20);RAISE EXCEPTION 'Invalid page accepted';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 r:=public.backend_member_build_list(actor,2,1);
 IF r->>'total'<>'1' OR r->'items'<>'[]'::jsonb THEN RAISE EXCEPTION 'Pagination total lost';END IF;
 r:=public.backend_member_build_patch(actor,v_id,'{"items":[]}');
 IF r->'items'<>'[]'::jsonb OR r->'ram_items'<>original->'ram_items' THEN RAISE EXCEPTION 'Clear affected omitted RAM';END IF;
 -- Restore all original non-RAM requirements, never infer one per category.
 r:=public.backend_member_build_patch(actor,v_id,jsonb_build_object('items',payload->'items'));
 IF r->'items'<>original->'items' THEN RAISE EXCEPTION 'Restore lost requirements';END IF;
 r:=public.backend_member_build_create(actor,'{"name":"Temporary empty draft","items":[],"ram_items":[]}');temp_id:=(r->>'id')::uuid;
 PERFORM public.backend_member_build_delete(actor,temp_id);
 BEGIN PERFORM public.backend_member_build_get(actor,temp_id);RAISE EXCEPTION 'Deleted draft readable';EXCEPTION WHEN no_data_found THEN NULL;END;
 PERFORM public.backend_member_build_create(other_actor,'{"name":"Admin own draft","items":[],"ram_items":[]}');
 IF public.backend_member_build_list(other_actor,1,20)->>'total'<>'1' THEN RAISE EXCEPTION 'Admin own draft unavailable';END IF;
END $$;
RESET ROLE;
DROP TRIGGER test_draft_failure ON public.build_ram_items;
DROP FUNCTION public.test_draft_write_failure();
DO $$ BEGIN
 BEGIN DELETE FROM public.products WHERE id='33333333-3333-4333-8333-333333333333';RAISE EXCEPTION 'Referenced product deleted';EXCEPTION WHEN foreign_key_violation THEN NULL;END;
 DELETE FROM auth.users WHERE id='11111111-1111-4111-8111-111111111111';
 IF NOT EXISTS(SELECT 1 FROM public.builds WHERE name='Renamed' AND owner_id IS NULL AND visibility='private') THEN RAISE EXCEPTION 'User deletion did not retain private anonymous draft';END IF;
 IF (SELECT count(*) FROM public.build_items i JOIN public.builds b ON b.id=i.build_id WHERE b.owner_id IS NULL)<>4 THEN RAISE EXCEPTION 'Account deletion dropped selections';END IF;
END $$;
SELECT 'PASS: member draft ownership, inactive/multi-item roundtrip, private policy, rollback, grants and account anonymization' AS draft_integration_result;
