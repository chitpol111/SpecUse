-- Phase-one catalog schema only. Apply manually to a separate local/test
-- Supabase database; the API executable does not execute migrations.
CREATE TABLE public.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    category TEXT NOT NULL CHECK (length(btrim(category)) > 0),
    brand TEXT NOT NULL CHECK (length(btrim(brand)) > 0),
    price_thb BIGINT NOT NULL CHECK (price_thb >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX products_active_category_id_idx
    ON public.products (category, id) WHERE is_active = TRUE;

-- Default-deny direct access through the Supabase Data API. No anon/member
-- policies are added in this phase. A future backend adapter must still
-- explicitly filter active products if its database role bypasses RLS.
ALTER TABLE public.products ENABLE ROW LEVEL SECURITY;

-- TODO(FE/BE): agree category values and compatibility JSON field names.
-- TODO: add typed CPU/motherboard/RAM tables in the compatibility phase.
-- TODO: settle final RLS policies and backend database access strategy.
