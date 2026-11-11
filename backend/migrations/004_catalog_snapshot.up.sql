-- Canonical hardware metadata remains nullable and is never backfilled from
-- the synthetic fixture. Legacy CPU/MB/RAM data stays in its typed tables.
ALTER TABLE public.products ADD COLUMN hardware_specs jsonb;
ALTER TABLE public.products ADD CONSTRAINT products_hardware_specs_shape CHECK (
 hardware_specs IS NULL OR
 (category IN ('gpu','storage','motherboard') AND jsonb_typeof(hardware_specs)='object')
);
REVOKE ALL ON public.products,public.product_cpus,public.product_motherboards,
public.ram_models,public.product_rams FROM PUBLIC,anon,authenticated,service_role;

CREATE FUNCTION public.backend_product_json(p_id uuid) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path='' AS $$
 SELECT jsonb_build_object('id',p.id,'name',p.name,'category',p.category,'brand',p.brand,
 'price_thb',p.price_thb,'is_active',p.is_active)
 || CASE p.category
 WHEN 'cpu' THEN jsonb_build_object('specs',jsonb_build_object('cpu_socket',c.cpu_socket))
 WHEN 'motherboard' THEN jsonb_build_object('specs',jsonb_build_object('cpu_socket',m.cpu_socket,'ram_type',m.ram_type)||coalesce(p.hardware_specs,'{}'::jsonb))
 WHEN 'ram' THEN jsonb_build_object('ram_model_id',r.ram_model_id,'specs',jsonb_build_object('ram_type',r.ram_type,'modules_per_package',r.modules_per_package))
 WHEN 'gpu' THEN jsonb_build_object('specs',p.hardware_specs)
 WHEN 'storage' THEN jsonb_build_object('specs',p.hardware_specs)
 ELSE '{}'::jsonb END
 FROM public.products p
 LEFT JOIN public.product_cpus c ON c.product_id=p.id
 LEFT JOIN public.product_motherboards m ON m.product_id=p.id
 LEFT JOIN public.product_rams r ON r.product_id=p.id WHERE p.id=p_id
$$;
CREATE FUNCTION public.backend_catalog_snapshot() RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path='' AS $$
 SELECT jsonb_build_object(
 'products',coalesce((SELECT jsonb_agg(public.backend_product_json(p.id) ORDER BY p.id) FROM public.products p),'[]'::jsonb),
 'ram_models',coalesce((SELECT jsonb_agg(jsonb_build_object('id',m.id,'ram_type',m.ram_type) ORDER BY m.id) FROM public.ram_models m),'[]'::jsonb))
$$;
REVOKE ALL ON FUNCTION public.backend_product_json(uuid),public.backend_catalog_snapshot() FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.backend_catalog_snapshot() TO service_role;
-- No direct anonymous/member RLS policy: the browser talks only to Go.
-- Snapshot is one SQL statement (one MVCC view), not separate table fetches.
