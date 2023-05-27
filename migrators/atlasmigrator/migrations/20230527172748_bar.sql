-- Create "cats" table
CREATE TABLE public.cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name text
);
