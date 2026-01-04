.PHONY: help clean build test run-test-mode e2e-clean e2e-up e2e-down e2e
.PHONY: git-diff-cached git-commit-with-editor git-show

help:
	@echo "Targets:"; \
	echo "  clean         Remove built binary and wipe ./_tmp/data (via helper container)"; \
	echo "  build         Build ./ssh-bastion"; \
	echo "  test          Run Go unit tests"; \
	echo "  run-test-mode Run HTTP+DNS locally with override identity"; \
	echo "  e2e-clean     Wipe bind-mounted /data contents (debug helper)"; \
	echo "  e2e-up        docker compose up -d --build --force-recreate (debug helper)"; \
	echo "  e2e-down      docker compose down (debug helper)"; \
	echo "  e2e           Run all e2e/scripts/e2e-NN-*.sh scenarios"

git-diff-cached:
	git --no-pager diff --cached

git-commit-with-editor:
	git -c core.editor='code --wait' commit -v -e -F $(lastword $(sort $(wildcard _tmp/git-commit/*.txt)))

git-show:
	git --no-pager show -1 --name-status --pretty=fuller && git status

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
	@# Run HTTP + DNS locally with override identity (no auth proxy required).
	@echo "Starting ssh-bastion (test mode)"
	@echo "Open: http://localhost:8080/"
	@echo "DNS:  udp://127.0.0.1:5353"
	@echo "Try:  dig @127.0.0.1 -p 5353 example.com A +short"
	@echo "Stop: Ctrl+C"
	@mkdir -p ./_tmp/data
	@# DNS upstream is auto-detected by ssh-bastion when not set:
	@# -dns-upstream flag > SSHBASTION_DNS_UPSTREAM env > /etc/resolv.conf (first nameserver)
	@SSHBASTION_DATA_DIR="_tmp/data" \
	SSHBASTION_AUTH_MODE="easy_auth" \
	SSHBASTION_AUTH_OVERRIDE_USER_ID="test-user-123" \
	SSHBASTION_AUTH_OVERRIDE_EMAIL="developer@localhost" \
	./ssh-bastion serve -http=true -dns=true -dns-addr ":5353"

e2e-clean:
	@# Debug helper: wipe bind-mounted /data contents.
	@mkdir -p ./_tmp/data
	@docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

e2e-up:
	@# Debug helper: bring up compose services.
	docker compose up -d --build --force-recreate

e2e-down:
	@# Debug helper: bring down compose services.
	docker compose down

e2e:
	@# Run all numbered end-to-end scenarios.
	@bash ./e2e/scripts/run.sh
