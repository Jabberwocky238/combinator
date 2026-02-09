RDB_TAGS=rdb_psql
KV_TAGS=kv_tikv,kv_redis
S3_TASGS=s3_minio,s3_seaweedfs

# BUILD_TAGS=$(RDB_TAGS),$(KV_TAGS),$(S3_TASGS)
# BUILD_TAGS=kv_rocksdb,rdb_psql

# 判断系统类型设置后缀
ifeq ($(OS),Windows_NT)
	SUFFIX=.exe
else
	SUFFIX=
endif

dev:
	go run -tags=dev,$(BUILD_TAGS) ./cmd dev

build-dev:
	go build -tags=dev,$(BUILD_TAGS) -o bin/dev$(SUFFIX) ./cmd

build-prod:
	go build -tags=prod -o bin/combinator$(SUFFIX) ./cmd

.PHONY: dev build-dev build-prod

test-migrate:
	go run -tags=$(BUILD_TAGS) ./cmd run migrate -i 0 -d ./tests/migrations

test-migrate-psql:
	go run -tags=$(BUILD_TAGS) ./cmd run migrate -i 1 -d ./tests/migrations