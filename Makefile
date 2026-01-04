.PHONY: help clean build test run-test-mode e2e-clean e2e-up e2e-down e2e
.PHONY: git-diff-cached git-commit-with-editor git-show

# run-test-mode variables (override at invocation time)
# Example:
#   make run-test-mode ID=test-user-123 EMAIL=developer@localhost ROLE=admin ADMINS=test-user-123,test-user-456
#   make run-test-mode DATA_DIR=_tmp/data DNS_UPSTREAM=127.0.0.11:53
ID ?= test-user-123
EMAIL ?= developer@localhost
ROLE ?= user
ADMINS ?=
DATA_DIR ?= _tmp/data
DNS_UPSTREAM ?=

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
	@echo "  HTTP:     :8080/tcp (Open in browser at http://localhost:8080/)"
	@echo "  DNS:      :5353/udp (Query via dig @localhost -p 5353 example.com A +short)"
	@echo "  Identity: ID=$(ID) EMAIL=$(EMAIL)"
	@echo "  Roles:    ROLE=$(ROLE) ADMINS=$(ADMINS)"
	@echo "  Data:     DATA_DIR=$(DATA_DIR)"
	@echo "  DNS:      DNS_UPSTREAM=$(DNS_UPSTREAM)"
	@echo "Type Ctrl-C to stop."
	@mkdir -p "$(DATA_DIR)"
	@SSHBASTION_DATA_DIR="$(DATA_DIR)" \
	SSHBASTION_DNS_UPSTREAM="$(DNS_UPSTREAM)" \
	SSHBASTION_AUTH_MODE="easy_auth" \
	SSHBASTION_AUTH_OVERRIDE_USER_ID="$(ID)" \
	SSHBASTION_AUTH_OVERRIDE_EMAIL="$(EMAIL)" \
	SSHBASTION_ROLE_DEFAULT="$(ROLE)" \
	SSHBASTION_ROLE_ADMIN_IDS="$(ADMINS)" \
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
