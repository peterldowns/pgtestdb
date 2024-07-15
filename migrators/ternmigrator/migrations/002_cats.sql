CREATE TABLE cats (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	name text
);

---- create above / drop below ----

DROP TABLE cats;
