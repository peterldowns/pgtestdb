-- migrate:up
CREATE TABLE public.cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name text
);

-- migrate:down
DROP TABLE public.cats;