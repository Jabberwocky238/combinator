BUILD_TAGS=rdb_psql
# BUILD_TAGS=kv_rocksdb,rdb_psql

dev:
	go run -tags=$(BUILD_TAGS) cmd/main.go

build:
	go build -tags=$(BUILD_TAGS) -o bin/combinator cmd/main.go

.PHONY: dev build
