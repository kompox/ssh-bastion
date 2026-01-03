.PHONY: clean build test run-test-mode e2e-clean e2e-up e2e-down e2e

clean:
	rm -f ./ssh-bastion
	@# With bind mounts, files may be owned by root (written from containers).
	@# Clean via a helper container first so host cleanup is reliable.
	@mkdir -p ./_tmp/data
	@docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*' >/dev/null 2>&1 || true
	rm -rf ./_tmp/data

build:
	go build -o ./ssh-bastion ./cmd/ssh-bastion

test:
	go test -v ./...

run-test-mode: build
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
	@mkdir -p ./_tmp/data
	@docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

e2e-up:
	docker compose up -d --build

e2e-down:
	docker compose down

e2e:
	@bash ./e2e/scripts/run.sh
