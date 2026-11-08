.PHONY: build run test lint migrate release-linux clean

# 构建二进制；SQLite 使用纯 Go 驱动，不需要 CGO。
build:
	go build -o bin/harmony ./cmd/harmony

# 本机 Linux 交叉编译（上线用）。
release-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -X main.version=$$(git describe --tags --always)" \
		-o bin/harmony-linux-amd64 ./cmd/harmony

# 运行（使用本机配置）。
run:
	go run ./cmd/harmony -config configs/config-local.yaml

# 新建/补齐 SQLite 表（快照全部 IF NOT EXISTS，可重复执行）。
migrate:
	go run ./cmd/harmony migrate -config configs/config-local.yaml

# 运行测试（每个测试使用临时 SQLite 文件）。
test:
	go test ./... -count=1

# 代码检查
lint:
	go vet ./...
	goimports -w $$(find . -name '*.go' -not -path './vendor/*')

# 清理
clean:
	rm -rf bin/
