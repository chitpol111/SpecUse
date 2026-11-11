-- Apply after 001 in a local/test database only. No catalog values are guessed
-- or backfilled; existing CPU/motherboard/RAM rows need trusted metadata.
-- Existing uppercase categories cause a constraint error instead of a silent rewrite.
ALTER TABLE public.products ADD CONSTRAINT products_category_lowercase
    CHECK (category = lower(category) AND category = btrim(category));
ALTER TABLE public.products ADD CONSTRAINT products_id_category_unique UNIQUE (id, category);

CREATE TABLE public.product_cpus (
    product_id UUID PRIMARY KEY,
    category TEXT NOT NULL DEFAULT 'cpu' CHECK (category = 'cpu'),
    cpu_socket TEXT NOT NULL CHECK (length(btrim(cpu_socket)) > 0),
    FOREIGN KEY (product_id, category) REFERENCES public.products (id, category) ON DELETE CASCADE
);

CREATE TABLE public.product_motherboards (
    product_id UUID PRIMARY KEY,
    category TEXT NOT NULL DEFAULT 'motherboard' CHECK (category = 'motherboard'),
    cpu_socket TEXT NOT NULL CHECK (length(btrim(cpu_socket)) > 0),
    ram_type TEXT NOT NULL CHECK (length(btrim(ram_type)) > 0),
    FOREIGN KEY (product_id, category) REFERENCES public.products (id, category) ON DELETE CASCADE
);

CREATE TABLE public.ram_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ram_type TEXT NOT NULL CHECK (length(btrim(ram_type)) > 0),
    UNIQUE (id, ram_type)
);

CREATE TABLE public.product_rams (
    product_id UUID PRIMARY KEY,
    category TEXT NOT NULL DEFAULT 'ram' CHECK (category = 'ram'),
    ram_model_id UUID NOT NULL,
    ram_type TEXT NOT NULL CHECK (length(btrim(ram_type)) > 0),
    modules_per_package SMALLINT NOT NULL CHECK (modules_per_package IN (1, 2)),
    FOREIGN KEY (product_id, category) REFERENCES public.products (id, category) ON DELETE CASCADE,
    FOREIGN KEY (ram_model_id, ram_type) REFERENCES public.ram_models (id, ram_type)
);

CREATE INDEX product_rams_model_idx ON public.product_rams (ram_model_id);

-- Keep direct Data API access default-deny pending the final RLS design.
ALTER TABLE public.product_cpus ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.product_motherboards ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.ram_models ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.product_rams ENABLE ROW LEVEL SECURITY;

-- TODO: build persistence/publication constraints follow the completeness
-- decision. Pairwise compatibility alone is not permission to publish.
