BEGIN;

CREATE FUNCTION public.backend_product_save_checked(
 p_actor uuid, p_product jsonb, p_create boolean, p_expected jsonb
) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE v_id uuid;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM public.profiles WHERE id=p_actor AND role='admin') THEN
  RAISE EXCEPTION 'Admin required' USING ERRCODE='42501';
 END IF;
 IF NOT p_create THEN
  IF p_expected IS NULL OR jsonb_typeof(p_expected) <> 'object' THEN
   RAISE EXCEPTION 'Expected product required' USING ERRCODE='22023';
  END IF;
  v_id := (p_product->>'id')::uuid;
  PERFORM 1 FROM public.products WHERE id=v_id FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Product not found' USING ERRCODE='P0002'; END IF;
  IF public.backend_product_json(v_id) IS DISTINCT FROM p_expected THEN
   RAISE EXCEPTION 'Product changed; reload before saving' USING ERRCODE='PT409';
  END IF;
 END IF;
 RETURN public.backend_product_save(p_actor,p_product,p_create);
END $$;

REVOKE ALL ON FUNCTION public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)
 FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.backend_product_save_checked(uuid,jsonb,boolean,jsonb)
 TO service_role;

COMMIT;
