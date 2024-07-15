# migrators

pgtestdb works with any migration framework, and includes out-of-the-box
adapters for the most popular golang frameworks:

- [pgmigrator](migrators/pgmigrator/) for [peterldowns/pgmigrate](https://github.com/peterldowns/pgmigrate)
- [golangmigrator](migrators/golangmigrator/) for [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
- [goosemigrator](migrators/goosemigrator/) for [pressly/goose](https://github.com/pressly/goose)
- [dbmatemigrator](migrators/dbmatemigrator/) for [amacneil/dbmate](https://github.com/amacneil/dbmate)
- [atlasmigrator](migrators/atlasmigrator/) for [ariga/atlas](https://github.com/ariga/atlas)
- [sqlmigrator](migrators/sqlmigrator/) for [rubenv/sql-migrate](https://github.com/rubenv/sql-migrate)
- [bunmigrator](migrators/bunmigrator/) for [uptrace/bun](https://github.com/uptrace/bun) (contributed by [@BrynBerkeley](https://github.com/BrynBerkeley))

If you're writing your own `Migrator`, I recommend you use the existing ones
as examples. Most migrators need to do some kind of file/directory hashing in
order to implement `Hash()` &mdash; you may want to use [the helpers in the
`common` subpackage](./common).
