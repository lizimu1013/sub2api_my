.PHONY: build build-backend build-frontend build-playground build-datamanagementd test test-backend test-frontend test-frontend-critical test-datamanagementd secret-scan

# gpt_image_playground 子项目位置（默认为同级目录，可通过环境变量覆盖）
PLAYGROUND_DIR ?= ../gpt_image_playground_my
PLAYGROUND_OUT := frontend/public/playground

FRONTEND_CRITICAL_VITEST := \
	src/views/auth/__tests__/LinuxDoCallbackView.spec.ts \
	src/views/auth/__tests__/WechatCallbackView.spec.ts \
	src/views/user/__tests__/PaymentView.spec.ts \
	src/views/user/__tests__/PaymentResultView.spec.ts \
	src/components/user/profile/__tests__/ProfileInfoCard.spec.ts \
	src/views/admin/__tests__/SettingsView.spec.ts

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
# build-playground 必须在 build-frontend 之前完成，因为 playground 产物会被
# 放进 frontend/public/playground/，由 Vite 拷贝进最终的 dist/。
build-frontend: build-playground
	@pnpm --dir frontend run build

# 编译外部 gpt_image_playground 子项目并把产物拷到 frontend/public/playground/
# 通过 VITE_BASE_PATH=/playground/ 让其资源路径与主站静态挂载一致。
# 若 PLAYGROUND_DIR 不存在则跳过（允许主站独立构建，例如 CI 上只有 sub2api 仓库）。
build-playground:
	@if [ -d "$(PLAYGROUND_DIR)" ]; then \
		echo ">> Building gpt_image_playground from $(PLAYGROUND_DIR)"; \
		cd "$(PLAYGROUND_DIR)" && VITE_BASE_PATH=/playground/ npm run build; \
		rm -rf "$(CURDIR)/$(PLAYGROUND_OUT)"; \
		mkdir -p "$(CURDIR)/$(PLAYGROUND_OUT)"; \
		cp -R "$(PLAYGROUND_DIR)/dist/." "$(CURDIR)/$(PLAYGROUND_OUT)/"; \
		echo ">> Playground assets copied to $(PLAYGROUND_OUT)"; \
	else \
		echo ">> Skipping playground build: $(PLAYGROUND_DIR) not found"; \
	fi

# 编译 datamanagementd（宿主机数据管理进程）
build-datamanagementd:
	@cd datamanagement && go build -o datamanagementd ./cmd/datamanagementd

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck
	@$(MAKE) test-frontend-critical

test-frontend-critical:
	@pnpm --dir frontend exec vitest run $(FRONTEND_CRITICAL_VITEST)
