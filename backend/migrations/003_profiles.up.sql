-- Run as project owner, after 001/002, on a separate local/test project first.
-- Public clients cannot read/write these tables or invoke backend RPCs.
CREATE TABLE public.profiles (
 id uuid PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
 display_name text NOT NULL CHECK (char_length(btrim(display_name)) BETWEEN 1 AND 100),
 role text NOT NULL DEFAULT 'member' CHECK (role IN ('member','admin')),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE public.profiles ENABLE ROW LEVEL SECURITY;
REVOKE ALL ON public.profiles FROM PUBLIC, anon, authenticated, service_role;

CREATE FUNCTION public.backend_new_user_profile() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
BEGIN
 INSERT INTO public.profiles(id,display_name,role)
 VALUES(NEW.id,coalesce(nullif(left(btrim(NEW.raw_user_meta_data->>'display_name'),100),''),'Member'),'member');
 RETURN NEW;
END $$;
REVOKE ALL ON FUNCTION public.backend_new_user_profile() FROM PUBLIC,anon,authenticated;
CREATE TRIGGER backend_create_profile AFTER INSERT ON auth.users
FOR EACH ROW EXECUTE FUNCTION public.backend_new_user_profile();
-- Existing accounts are initialized without reading a client-supplied role.
INSERT INTO public.profiles(id,display_name,role)
SELECT id,coalesce(nullif(left(btrim(raw_user_meta_data->>'display_name'),100),''),'Member'),'member'
FROM auth.users ON CONFLICT(id) DO NOTHING;

CREATE FUNCTION public.backend_profile_get(p_user_id uuid) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE result jsonb;
BEGIN
 SELECT to_jsonb(p) INTO result FROM public.profiles p WHERE id=p_user_id;
 IF result IS NULL THEN RAISE EXCEPTION 'Profile unavailable' USING ERRCODE='P0002'; END IF;
 RETURN result;
END $$;
CREATE FUNCTION public.backend_profile_update(p_user_id uuid,p_display_name text) RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path='' AS $$
DECLARE result jsonb;
BEGIN
 IF p_display_name IS NULL OR char_length(btrim(p_display_name)) NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'Invalid display name' USING ERRCODE='23514';
 END IF;
 UPDATE public.profiles SET display_name=btrim(p_display_name),updated_at=now()
 WHERE id=p_user_id RETURNING to_jsonb(profiles) INTO result;
 IF result IS NULL THEN RAISE EXCEPTION 'Profile unavailable' USING ERRCODE='P0002'; END IF;
 RETURN result;
END $$;
REVOKE ALL ON FUNCTION public.backend_profile_get(uuid),public.backend_profile_update(uuid,text) FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.backend_profile_get(uuid),public.backend_profile_update(uuid,text) TO service_role;
