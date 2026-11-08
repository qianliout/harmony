.PHONY: release release-linux build-h5 copy-web build-backend release-mp deploy migrate dev-backend dev-h5

# 交付：单个二进制内置 H5 产物，浏览器打开 http://localhost:8080 即可使用。
release: build-h5 copy-web build-backend

build-h5:
	cd frontend && pnpm install && pnpm build:h5

copy-web:
	rm -rf backend/web/dist
	mkdir -p backend/web
	cp -r frontend/dist/build/h5 backend/web/dist

build-backend:
	$(MAKE) -C backend build

# 上线：本机交叉编译 Linux 二进制，H5 同时打进二进制。
release-linux: build-h5 copy-web
	$(MAKE) -C backend release-linux

# 上线：上传二进制、备份、迁移、重启、健康检查，失败自动回滚。
deploy: release-linux
	HOST=$(HOST) ./deploy/deploy.sh

# 小程序：先构建，再确认产物里没有识别相关代码。
migrate:
	$(MAKE) -C backend migrate

release-mp:
	cd frontend && pnpm build:mp-weixin
	@if grep -rniE "recognize|识别" frontend/dist/build/mp-weixin --include=*.js --include=*.json --include=*.wxml; then \
		echo "小程序产物里出现了识别相关代码"; exit 1; \
	fi

dev-backend:
	$(MAKE) -C backend run

dev-h5:
	cd frontend && pnpm dev:h5
