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
	@UPSTREAM_IP=$$(awk '/^nameserver/{print $$2; exit}' /etc/resolv.conf); \
	if [ -z "$$UPSTREAM_IP" ]; then \
		echo "ERROR: Could not detect DNS upstream from /etc/resolv.conf" >&2; \
		exit 1; \
	fi; \
	case "$$UPSTREAM_IP" in \
		*:*) UPSTREAM="[$$UPSTREAM_IP]:53" ;; \
		*)   UPSTREAM="$$UPSTREAM_IP:53" ;; \
	esac; \
	SSHBASTION_DATA_DIR="_tmp/data" \
	SSHBASTION_AUTH_MODE="easy_auth" \
	SSHBASTION_AUTH_OVERRIDE_USER_ID="test-user-123" \
	SSHBASTION_AUTH_OVERRIDE_EMAIL="developer@localhost" \
	SSHBASTION_DNS_UPSTREAM="$$UPSTREAM" \
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
	@set -eu; \
	if command -v ss >/dev/null 2>&1; then \
		in_use=0; \
		check_tcp() { \
			p="$$1"; \
			if ss -H -lnt | awk -v p=":$${p}$$" '$$4 ~ p {exit 0} END{exit 1}'; then \
				echo "ERROR: TCP port $${p} is already in use (E2E requires localhost:$${p})." >&2; \
				in_use=1; \
			fi; \
		}; \
		check_udp() { \
			p="$$1"; \
			if ss -H -lnu | awk -v p=":$${p}$$" '$$4 ~ p {exit 0} END{exit 1}'; then \
				echo "ERROR: UDP port $${p} is already in use (E2E requires 127.0.0.1:$${p}/udp)." >&2; \
				in_use=1; \
			fi; \
		}; \
		check_tcp 8080; \
		check_tcp 2222; \
		check_udp 5353; \
		if [ "$$in_use" -ne 0 ]; then \
			echo "Hint: stop any local dev servers (e.g. make run-test-mode) before running E2E." >&2; \
			exit 1; \
		fi; \
	else \
		echo "WARN: ss not found; skipping E2E port preflight check" >&2; \
	fi; \
	for f in $$(ls -1 ./e2e/scripts/e2e-[0-9][0-9]-*.sh | LC_ALL=C sort); do \
		case "$$f" in \
			*'-skip-'*.sh|*'-xfail-'*.sh|*'-known-fail-'*.sh|*'-quarantine-'*.sh) \
				echo "==> SKIP $$f"; \
				;; \
			*) \
				echo "==> $$f"; \
				bash "$$f"; \
				;; \
		esac; \
	done
