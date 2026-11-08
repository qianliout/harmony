.PHONY: release build-h5 copy-web build-backend release-mp dev-backend dev-h5

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

# 小程序：先构建，再确认产物里没有识别相关代码。
release-mp:
	cd frontend && pnpm build:mp-weixin
	@if grep -rniE "recognize|识别" frontend/dist/build/mp-weixin --include=*.js --include=*.json --include=*.wxml; then \
		echo "小程序产物里出现了识别相关代码"; exit 1; \
	fi

dev-backend:
	$(MAKE) -C backend run

dev-h5:
	cd frontend && pnpm dev:h5
