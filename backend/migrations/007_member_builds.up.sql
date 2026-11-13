-- Additive migration after 001–006. Inspect migration history first.
-- Do not apply automatically to specuse-dev or rerun earlier migrations.
BEGIN;
CREATE TABLE public.builds (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid REFERENCES public.profiles(id) ON DELETE SET NULL,
 name text NOT NULL CHECK(char_length(btrim(name)) BETWEEN 1 AND 100),
 description text NOT NULL DEFAULT '' CHECK(char_length(description)<=2000),
 visibility text NOT NULL DEFAULT 'private' CHECK(visibility IN ('private','public')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(owner_id IS NOT NULL OR visibility='private')
);
CREATE INDEX builds_owner_updated_idx ON public.builds(owner_id,updated_at DESC,id);
CREATE TABLE public.build_items (
 build_id uuid NOT NULL REFERENCES public.builds(id) ON DELETE CASCADE,
 product_id uuid NOT NULL REFERENCES public.products(id) ON DELETE RESTRICT,
 quantity bigint NOT NULL CHECK(quantity>0),
 owned_quantity bigint NOT NULL CHECK(owned_quantity>=0 AND owned_quantity<=quantity),
 position bigint NOT NULL CHECK(position>=1),
 PRIMARY KEY(build_id,product_id), UNIQUE(build_id,position)
);
CREATE TABLE public.build_ram_items (
 build_id uuid NOT NULL REFERENCES public.builds(id) ON DELETE CASCADE,
 ram_model_id uuid NOT NULL REFERENCES public.ram_models(id) ON DELETE RESTRICT,
 required_modules bigint NOT NULL CHECK(required_modules>0),
 owned_modules bigint NOT NULL CHECK(owned_modules>=0 AND owned_modules<=required_modules),
 position bigint NOT NULL CHECK(position>=1),
 PRIMARY KEY(build_id,ram_model_id), UNIQUE(build_id,position)
);
ALTER TABLE public.builds ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.build_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.build_ram_items ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.builds,public.build_items,public.build_ram_items FROM PUBLIC,anon,authenticated,service_role;

-- Even privileged RPC callers cannot publish before a publication policy exists.
-- Auth user deletion cascades to profiles, anonymizes builds and preserves rows.
CREATE FUNCTION public.backend_member_build_private() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 IF NEW.owner_id IS NULL THEN NEW.visibility:='private';END IF;
 IF NEW.visibility<>'private' THEN RAISE EXCEPTION 'Public builds disabled' USING ERRCODE='23514';END IF;
 IF TG_OP='UPDATE' THEN NEW.updated_at:=clock_timestamp();END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER backend_private_build BEFORE INSERT OR UPDATE ON public.builds
FOR EACH ROW EXECUTE FUNCTION public.backend_member_build_private();

CREATE FUNCTION public.backend_member_actor(p_actor uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 PERFORM 1 FROM public.profiles WHERE id=p_actor AND role IN ('member','admin') FOR KEY SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Member required' USING ERRCODE='42501';END IF;
END $$;

-- Validates writes independently of the Go boundary. Inactive references are
-- allowed for drafts; compatibility, active-pack selection and pricing stay in
-- the existing validate API and are never accepted as client truth.
CREATE FUNCTION public.backend_member_build_check(p_data jsonb,p_create boolean) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE r jsonb;v_id uuid;v_category text;v_required bigint;v_owned bigint;
BEGIN
 IF p_data IS NULL OR jsonb_typeof(p_data)<>'object' OR p_data='{}'::jsonb THEN
  RAISE EXCEPTION 'Invalid build fields' USING ERRCODE='22023';END IF;
 IF EXISTS(SELECT 1 FROM jsonb_object_keys(p_data) AS k WHERE k NOT IN ('name','description','visibility','items','ram_items')) THEN
  RAISE EXCEPTION 'Unknown build field' USING ERRCODE='22023';END IF;
 IF p_create AND NOT(p_data ?& ARRAY['name','items','ram_items']) THEN
  RAISE EXCEPTION 'Missing build fields' USING ERRCODE='22023';END IF;
 IF p_data?'name' AND (jsonb_typeof(p_data->'name') IS DISTINCT FROM 'string' OR char_length(btrim(p_data->>'name')) NOT BETWEEN 1 AND 100) THEN
  RAISE EXCEPTION 'Invalid name' USING ERRCODE='22023';END IF;
 IF p_data?'description' AND (jsonb_typeof(p_data->'description') IS DISTINCT FROM 'string' OR char_length(p_data->>'description')>2000) THEN
  RAISE EXCEPTION 'Invalid description' USING ERRCODE='22023';END IF;
 IF p_data?'visibility' THEN
  IF p_data->>'visibility'='public' THEN RAISE EXCEPTION 'Public builds disabled' USING ERRCODE='23514';END IF;
  IF jsonb_typeof(p_data->'visibility') IS DISTINCT FROM 'string' OR p_data->>'visibility'<>'private' THEN
   RAISE EXCEPTION 'Invalid visibility' USING ERRCODE='22023';END IF;
 END IF;
 IF p_data?'items' THEN
  IF jsonb_typeof(p_data->'items') IS DISTINCT FROM 'array' THEN RAISE EXCEPTION 'Invalid items' USING ERRCODE='22023';END IF;
  FOR r IN SELECT value FROM jsonb_array_elements(p_data->'items') LOOP
   IF jsonb_typeof(r) IS DISTINCT FROM 'object' OR NOT(r ?& ARRAY['product_id','quantity','owned_quantity']) THEN
    RAISE EXCEPTION 'Invalid item fields' USING ERRCODE='22023';END IF;
   IF EXISTS(SELECT 1 FROM jsonb_object_keys(r) AS k WHERE k NOT IN ('product_id','quantity','owned_quantity')) THEN
    RAISE EXCEPTION 'Unknown item field' USING ERRCODE='22023';END IF;
   IF jsonb_typeof(r->'product_id') IS DISTINCT FROM 'string' OR (r->>'product_id') !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
     OR jsonb_typeof(r->'quantity') IS DISTINCT FROM 'number' OR (r->>'quantity') !~ '^[0-9]+$'
     OR jsonb_typeof(r->'owned_quantity') IS DISTINCT FROM 'number' OR (r->>'owned_quantity') !~ '^[0-9]+$' THEN
    RAISE EXCEPTION 'Invalid item values' USING ERRCODE='22023';END IF;
   v_id:=(r->>'product_id')::uuid;v_required:=(r->>'quantity')::bigint;v_owned:=(r->>'owned_quantity')::bigint;
   IF v_required<1 OR v_owned>v_required THEN RAISE EXCEPTION 'Invalid item counts' USING ERRCODE='22023';END IF;
   SELECT category INTO v_category FROM public.products WHERE id=v_id FOR KEY SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Reference unavailable' USING ERRCODE='P0002';END IF;
   IF v_category='ram' THEN RAISE EXCEPTION 'RAM must use models' USING ERRCODE='22023';END IF;
  END LOOP;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_data->'items') AS x GROUP BY lower(x->>'product_id') HAVING count(*)>1) THEN
   RAISE EXCEPTION 'Duplicate product' USING ERRCODE='22023';END IF;
 END IF;
 IF p_data?'ram_items' THEN
  IF jsonb_typeof(p_data->'ram_items') IS DISTINCT FROM 'array' THEN RAISE EXCEPTION 'Invalid RAM items' USING ERRCODE='22023';END IF;
  FOR r IN SELECT value FROM jsonb_array_elements(p_data->'ram_items') LOOP
   IF jsonb_typeof(r) IS DISTINCT FROM 'object' OR NOT(r ?& ARRAY['ram_model_id','required_modules','owned_modules']) THEN
    RAISE EXCEPTION 'Invalid RAM fields' USING ERRCODE='22023';END IF;
   IF EXISTS(SELECT 1 FROM jsonb_object_keys(r) AS k WHERE k NOT IN ('ram_model_id','required_modules','owned_modules')) THEN
    RAISE EXCEPTION 'Unknown RAM field' USING ERRCODE='22023';END IF;
   IF jsonb_typeof(r->'ram_model_id') IS DISTINCT FROM 'string' OR (r->>'ram_model_id') !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
     OR jsonb_typeof(r->'required_modules') IS DISTINCT FROM 'number' OR (r->>'required_modules') !~ '^[0-9]+$'
     OR jsonb_typeof(r->'owned_modules') IS DISTINCT FROM 'number' OR (r->>'owned_modules') !~ '^[0-9]+$' THEN
    RAISE EXCEPTION 'Invalid RAM values' USING ERRCODE='22023';END IF;
   v_id:=(r->>'ram_model_id')::uuid;v_required:=(r->>'required_modules')::bigint;v_owned:=(r->>'owned_modules')::bigint;
   IF v_required<1 OR v_owned>v_required THEN RAISE EXCEPTION 'Invalid RAM counts' USING ERRCODE='22023';END IF;
   PERFORM 1 FROM public.ram_models WHERE id=v_id FOR KEY SHARE;
   IF NOT FOUND THEN RAISE EXCEPTION 'Reference unavailable' USING ERRCODE='P0002';END IF;
  END LOOP;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(p_data->'ram_items') AS x GROUP BY lower(x->>'ram_model_id') HAVING count(*)>1) THEN
   RAISE EXCEPTION 'Duplicate RAM model' USING ERRCODE='22023';END IF;
 END IF;
END $$;

CREATE FUNCTION public.backend_member_build_apply(p_id uuid,p_data jsonb) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 IF p_data?'items' THEN
  DELETE FROM public.build_items WHERE build_id=p_id;
  INSERT INTO public.build_items(build_id,product_id,quantity,owned_quantity,position)
  SELECT p_id,(value->>'product_id')::uuid,(value->>'quantity')::bigint,(value->>'owned_quantity')::bigint,ordinality
  FROM jsonb_array_elements(p_data->'items') WITH ORDINALITY;
 END IF;
 IF p_data?'ram_items' THEN
  DELETE FROM public.build_ram_items WHERE build_id=p_id;
  INSERT INTO public.build_ram_items(build_id,ram_model_id,required_modules,owned_modules,position)
  SELECT p_id,(value->>'ram_model_id')::uuid,(value->>'required_modules')::bigint,(value->>'owned_modules')::bigint,ordinality
  FROM jsonb_array_elements(p_data->'ram_items') WITH ORDINALITY;
 END IF;
END $$;

CREATE FUNCTION public.backend_member_build_json(p_id uuid) RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path='' AS $$
 SELECT jsonb_build_object('id',b.id,'name',b.name,'description',b.description,'visibility',b.visibility,'created_at',b.created_at,'updated_at',b.updated_at,
 'items',coalesce((SELECT jsonb_agg(jsonb_build_object('product_id',i.product_id,'quantity',i.quantity,'owned_quantity',i.owned_quantity,
  'product_summary',jsonb_build_object('name',p.name,'category',p.category,'is_active',p.is_active)) ORDER BY i.position)
  FROM public.build_items i JOIN public.products p ON p.id=i.product_id WHERE i.build_id=b.id),'[]'::jsonb),
 'ram_items',coalesce((SELECT jsonb_agg(jsonb_build_object('ram_model_id',i.ram_model_id,'required_modules',i.required_modules,'owned_modules',i.owned_modules,
  'ram_model_summary',jsonb_build_object('ram_type',m.ram_type),
  'active_package_count',(SELECT count(*) FROM public.product_rams r JOIN public.products p ON p.id=r.product_id WHERE r.ram_model_id=m.id AND p.is_active)) ORDER BY i.position)
  FROM public.build_ram_items i JOIN public.ram_models m ON m.id=i.ram_model_id WHERE i.build_id=b.id),'[]'::jsonb))
 FROM public.builds b WHERE b.id=p_id
$$;

CREATE FUNCTION public.backend_member_build_get(p_actor uuid,p_id uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE result jsonb;
BEGIN
 PERFORM public.backend_member_actor(p_actor);
 SELECT public.backend_member_build_json(b.id) INTO result FROM public.builds b WHERE b.id=p_id AND b.owner_id=p_actor;
 IF result IS NULL THEN RAISE EXCEPTION 'Build not found' USING ERRCODE='P0002';END IF;
 RETURN result;
END $$;
CREATE FUNCTION public.backend_member_build_list(p_actor uuid,p_page bigint,p_page_size bigint) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE result jsonb;v_offset bigint;
BEGIN
 PERFORM public.backend_member_actor(p_actor);
 IF p_page IS NULL OR p_page_size IS NULL OR p_page<1 OR p_page_size<1 THEN RAISE EXCEPTION 'Invalid page' USING ERRCODE='22023';END IF;
 v_offset:=(p_page-1)*p_page_size;
 WITH rows AS (SELECT * FROM public.builds WHERE owner_id=p_actor ORDER BY updated_at DESC,id LIMIT p_page_size OFFSET v_offset)
 SELECT jsonb_build_object('items',coalesce((SELECT jsonb_agg(jsonb_build_object('id',id,'name',name,'description',description,'visibility',visibility,'created_at',created_at,'updated_at',updated_at) ORDER BY updated_at DESC,id) FROM rows),'[]'::jsonb),
  'page',p_page,'page_size',p_page_size,'total',(SELECT count(*) FROM public.builds WHERE owner_id=p_actor)) INTO result;
 RETURN result;
END $$;
CREATE FUNCTION public.backend_member_build_create(p_actor uuid,p_build jsonb) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE v_id uuid;
BEGIN
 PERFORM public.backend_member_actor(p_actor);PERFORM public.backend_member_build_check(p_build,true);
 INSERT INTO public.builds(owner_id,name,description,visibility)
 VALUES(p_actor,btrim(p_build->>'name'),coalesce(p_build->>'description',''),'private') RETURNING id INTO v_id;
 PERFORM public.backend_member_build_apply(v_id,p_build);
 RETURN public.backend_member_build_json(v_id);
END $$;
CREATE FUNCTION public.backend_member_build_patch(p_actor uuid,p_id uuid,p_patch jsonb) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 PERFORM public.backend_member_actor(p_actor);
 PERFORM 1 FROM public.builds WHERE id=p_id AND owner_id=p_actor FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Build not found' USING ERRCODE='P0002';END IF;
 PERFORM public.backend_member_build_check(p_patch,false);
 UPDATE public.builds SET
  name=CASE WHEN p_patch?'name' THEN btrim(p_patch->>'name') ELSE name END,
  description=CASE WHEN p_patch?'description' THEN p_patch->>'description' ELSE description END
 WHERE id=p_id AND owner_id=p_actor;
 PERFORM public.backend_member_build_apply(p_id,p_patch);
 RETURN public.backend_member_build_json(p_id);
END $$;
CREATE FUNCTION public.backend_member_build_delete(p_actor uuid,p_id uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 PERFORM public.backend_member_actor(p_actor);
 DELETE FROM public.builds WHERE id=p_id AND owner_id=p_actor;
 IF NOT FOUND THEN RAISE EXCEPTION 'Build not found' USING ERRCODE='P0002';END IF;
 RETURN jsonb_build_object('deleted',true);
END $$;

REVOKE ALL ON FUNCTION public.backend_member_build_private(),public.backend_member_actor(uuid),
 public.backend_member_build_check(jsonb,boolean),public.backend_member_build_apply(uuid,jsonb),public.backend_member_build_json(uuid)
 FROM PUBLIC,anon,authenticated,service_role;
REVOKE ALL ON FUNCTION public.backend_member_build_get(uuid,uuid),public.backend_member_build_list(uuid,bigint,bigint),
 public.backend_member_build_create(uuid,jsonb),public.backend_member_build_patch(uuid,uuid,jsonb),public.backend_member_build_delete(uuid,uuid)
 FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.backend_member_build_get(uuid,uuid),public.backend_member_build_list(uuid,bigint,bigint),
 public.backend_member_build_create(uuid,jsonb),public.backend_member_build_patch(uuid,uuid,jsonb),public.backend_member_build_delete(uuid,uuid)
 TO service_role;

COMMIT;
