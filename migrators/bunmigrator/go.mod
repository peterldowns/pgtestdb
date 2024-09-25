module github.com/peterldowns/pgtestdb/migrators/bunmigrator

go 1.21

toolchain go1.22.1

replace github.com/peterldowns/pgtestdb => ../../

require (
	github.com/peterldowns/pgtestdb v0.0.14
	github.com/peterldowns/testy v0.0.1
	github.com/uptrace/bun v1.2.1
	github.com/uptrace/bun/dialect/pgdialect v1.2.1
	github.com/uptrace/bun/driver/pgdriver v1.2.1
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/sys v0.20.0 // indirect
	mellium.im/sasl v0.3.1 // indirect
)
