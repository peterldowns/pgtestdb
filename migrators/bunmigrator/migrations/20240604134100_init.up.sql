CREATE TABLE "public"."users" (
 "id" integer NOT NULL,
 "name" character varying(100) NULL,
 PRIMARY KEY ("id")
);

CREATE TABLE "public"."blog_posts" (
 "id" integer NOT NULL,
 "title" character varying(100) NULL,
 "body" text NULL,
 "author_id" integer NULL,
 PRIMARY KEY ("id"),
 CONSTRAINT "author_fk" FOREIGN KEY ("author_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);