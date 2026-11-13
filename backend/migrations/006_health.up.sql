CREATE FUNCTION public.backend_health() RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path='' AS $$ SELECT true $$;
REVOKE ALL ON FUNCTION public.backend_health() FROM PUBLIC,anon,authenticated;
GRANT EXECUTE ON FUNCTION public.backend_health() TO service_role;
