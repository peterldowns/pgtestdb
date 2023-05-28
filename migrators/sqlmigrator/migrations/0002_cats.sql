-- +migrate Up
CREATE TABLE public.cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name text
);

-- +migrate Down
DROP TABLE public.cats;
