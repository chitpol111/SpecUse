-- Server-only RPC mutations, with a second authoritative role check in DB.
CREATE FUNCTION public.backend_product_save(p_actor uuid,p_product jsonb,p_create boolean) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE v_id uuid; v_category text; v_specs jsonb;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM public.profiles WHERE id=p_actor AND role='admin') THEN
  RAISE EXCEPTION 'Admin required' USING ERRCODE='42501';
 END IF;
 v_id:=(p_product->>'id')::uuid;v_category:=p_product->>'category';v_specs:=nullif(p_product->'specs','null'::jsonb);
 IF p_create THEN
  INSERT INTO public.products(id,name,category,brand,price_thb,is_active)
  VALUES(v_id,p_product->>'name',v_category,p_product->>'brand',(p_product->>'price_thb')::bigint,(p_product->>'is_active')::boolean);
 ELSE
  PERFORM 1 FROM public.products WHERE id=v_id FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Product not found' USING ERRCODE='P0002';END IF;
  IF (SELECT category FROM public.products WHERE id=v_id)<>v_category THEN RAISE EXCEPTION 'Category immutable' USING ERRCODE='23514';END IF;
  UPDATE public.products SET name=p_product->>'name',brand=p_product->>'brand',price_thb=(p_product->>'price_thb')::bigint,is_active=(p_product->>'is_active')::boolean WHERE id=v_id;
 END IF;
 CASE v_category
 WHEN 'cpu' THEN
  INSERT INTO public.product_cpus(product_id,cpu_socket) VALUES(v_id,v_specs->>'cpu_socket')
  ON CONFLICT(product_id) DO UPDATE SET cpu_socket=excluded.cpu_socket;
 WHEN 'motherboard' THEN
  INSERT INTO public.product_motherboards(product_id,cpu_socket,ram_type) VALUES(v_id,v_specs->>'cpu_socket',v_specs->>'ram_type')
  ON CONFLICT(product_id) DO UPDATE SET cpu_socket=excluded.cpu_socket,ram_type=excluded.ram_type;
 WHEN 'ram' THEN
  INSERT INTO public.product_rams(product_id,ram_model_id,ram_type,modules_per_package)
  VALUES(v_id,(p_product->>'ram_model_id')::uuid,v_specs->>'ram_type',(v_specs->>'modules_per_package')::smallint)
  ON CONFLICT(product_id) DO UPDATE SET ram_model_id=excluded.ram_model_id,ram_type=excluded.ram_type,modules_per_package=excluded.modules_per_package;
 ELSE NULL;
 END CASE;
 UPDATE public.products SET hardware_specs=CASE
 WHEN v_category IN ('gpu','storage') THEN v_specs
 WHEN v_category='motherboard' THEN v_specs-'cpu_socket'-'ram_type'
 ELSE NULL END WHERE id=v_id;
 RETURN public.backend_product_json(v_id);
END $$;
CREATE FUNCTION public.backend_product_deactivate(p_actor uuid,p_id uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM public.profiles WHERE id=p_actor AND role='admin') THEN
  RAISE EXCEPTION 'Admin required' USING ERRCODE='42501';
 END IF;
 UPDATE public.products SET is_active=false WHERE id=p_id;
 IF NOT FOUND THEN RAISE EXCEPTION 'Product not found' USING ERRCODE='P0002';END IF;
 RETURN jsonb_build_object('deactivated',true);
END $$;
REVOKE ALL ON FUNCTION public.backend_product_save(uuid,jsonb,boolean),public.backend_product_deactivate(uuid,uuid) FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.backend_product_save(uuid,jsonb,boolean),public.backend_product_deactivate(uuid,uuid) TO service_role;
