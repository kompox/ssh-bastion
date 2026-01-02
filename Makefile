.PHONY: clean build test run-test-mode

clean:
	rm -f ./ssh-bastion
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
