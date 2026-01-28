BUILD_TAGS=rdb_psql
# BUILD_TAGS=kv_rocksdb,rdb_psql

dev:
	go run -tags=$(BUILD_TAGS) ./cmd start

build:
	go build -tags=$(BUILD_TAGS) -o bin/combinator ./cmd

.PHONY: dev build

test-migrate:
	go run -tags=$(BUILD_TAGS) ./cmd run migrate -i 0 -d ./tests/migrations

test-migrate-psql:
	go run -tags=$(BUILD_TAGS) ./cmd run migrate -i 1 -d ./tests/migrations