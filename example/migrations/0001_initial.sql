-- +goose Up
CREATE TABLE public.myhstoredata (
    h hstore,           -- requires "hstore" extension
    examplegeo geometry -- requires "postgis" extension
);

INSERT INTO public.myhstoredata VALUES ('name=>Peter, example=>yes');

INSERT INTO public.myhstoredata values 

-- +goose Down
DROP TABLE public.myhstoredata;