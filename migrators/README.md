# migrators

pgtestdb works with any migration framework, and includes out-of-the-box
adaptors for the most popular golang frameworks:

- [golangmigrator](./golangmigrator/) for [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [goosemigrator](./goosemigrator/) for [pressly/goose](https://github.com/pressly/goose)
- [dbmatemigrator](./dbmatemigrator/) for [amacneil/dbmate](https://github.com/amacneil/dbmate)
- [atlasmigrator](./atlasmigrator/) for [ariga/atlas](https://github.com/ariga/atlas)
- [sqlmigrator](./sqlmigrator/) for [rubenv/sql-migrate](https://github.com/rubenv/sql-migrate)

If you're writing your own `Migrator`, I recommend you use the existing ones
as examples. Most migrators need to do some kind of file/directory hashing in
order to implement `Hash()` &mdash; you may want to use [the helpers in the
`common` subpackage](./common).
