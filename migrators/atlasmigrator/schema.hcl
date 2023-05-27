schema "public" {
}

table "blog_posts" {
  schema = schema.public
  column "id" {
    null = false
    type = integer
  }
  column "title" {
    null = true
    type = character_varying(100)
  }
  column "body" {
    null = true
    type = text
  }
  column "author_id" {
    null = true
    type = integer
  }
  primary_key {
    columns = [column.id]
  }
  foreign_key "author_fk" {
    columns     = [column.author_id]
    ref_columns = [table.users.column.id]
    on_update   = NO_ACTION
    on_delete   = NO_ACTION
  }
}

table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = integer
  }
  column "name" {
    null = true
    type = character_varying(100)
  }
  primary_key {
    columns = [column.id]
  }
}

table "cats" {
  schema = schema.public
  column "id" {
    null = false
    type = bigint
    identity {
      generated = ALWAYS
    }
  }
  column "name" {
    null = true
    type = text
  }
  primary_key {
    columns = [column.id]
  }
}
