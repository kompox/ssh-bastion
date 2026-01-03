.PHONY: help clean build test run-test-mode e2e-clean e2e-up e2e-down e2e

help:
	@echo "Targets:"; \
	echo "  clean         Remove built binary and wipe ./_tmp/data (via helper container)"; \
	echo "  build         Build ./ssh-bastion"; \
	echo "  test          Run Go unit tests"; \
	echo "  run-test-mode Run web server locally with override identity"; \
	echo "  e2e-clean     Wipe bind-mounted /data contents (debug helper)"; \
	echo "  e2e-up        docker compose up -d --build (debug helper)"; \
	echo "  e2e-down      docker compose down (debug helper)"; \
	echo "  e2e           Run all e2e/scripts/e2e-NN-*.sh scenarios"

clean:
	@# Remove built binary and wipe persisted bind-mounted data.
	rm -f ./ssh-bastion
	@# With bind mounts, files may be owned by root (written from containers).
	@# Clean via a helper container first so host cleanup is reliable.
	@mkdir -p ./_tmp/data
	@docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*' >/dev/null 2>&1 || true
	rm -rf ./_tmp/data

build:
	@# Build the local Go binary.
	go build -o ./ssh-bastion ./cmd/ssh-bastion

test:
	@# Run unit tests.
	go test -v ./...

run-test-mode: build
	@# Run the web server locally with override identity (no auth proxy required).
	@echo "Starting ssh-bastion (test mode)"
	@echo "Open: http://localhost:8080/"
	@echo "Stop: Ctrl+C"
	@mkdir -p ./_tmp/data
	@SSHBASTION_DATA_DIR="_tmp/data" \
	SSHBASTION_AUTH_MODE="easy_auth" \
	SSHBASTION_AUTH_OVERRIDE_USER_ID="test-user-123" \
	SSHBASTION_AUTH_OVERRIDE_EMAIL="developer@localhost" \
	./ssh-bastion web

e2e-clean:
	@# Debug helper: wipe bind-mounted /data contents.
	@mkdir -p ./_tmp/data
	@docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

e2e-up:
	@# Debug helper: bring up compose services.
	docker compose up -d --build

e2e-down:
	@# Debug helper: bring down compose services.
	docker compose down

e2e:
	@# Run all numbered end-to-end scenarios.
	@set -eu; \
	for f in $$(ls -1 ./e2e/scripts/e2e-[0-9][0-9]-*.sh | LC_ALL=C sort); do \
		echo "==> $$f"; \
		bash "$$f"; \
	done
