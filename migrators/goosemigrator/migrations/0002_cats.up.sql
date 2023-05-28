-- +goose Up
CREATE TABLE public.cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name text
);
-- +goose Down
DROP TABLE public.cats;