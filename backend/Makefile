.PHONY: build run test lint clean

# 构建二进制；SQLite 驱动需要 CGO。
build:
	CGO_ENABLED=1 go build -o bin/harmony ./cmd/harmony

# 运行（使用本机配置）。
run:
	CGO_ENABLED=1 go run ./cmd/harmony -config configs/config-local.yaml

# 运行测试
test:
	CGO_ENABLED=1 go test ./... -count=1

# 代码检查
lint:
	CGO_ENABLED=1 go vet ./...
	CGO_ENABLED=1 goimports -w $(shell find . -name '*.go' -not -path './vendor/*')

# 清理
clean:
	rm -rf bin/
