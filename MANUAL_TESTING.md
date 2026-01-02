# SSH Bastion Web Application - Manual Testing

## Prerequisites

- Go 1.21 or later
- A directory for data storage

## Building

```bash
go build -o ssh-bastion ./cmd/ssh-bastion
```

## Running Locally

### 1. Create a data directory

```bash
mkdir -p _tmp/data
```

### 2. Set environment variables

```bash
export SSHBASTION_DATA_DIR="_tmp/data"
export SSHBASTION_AUTH_MODE="easy_auth"
```

### 3. Run the web server

```bash
./ssh-bastion web
```

The server will start on `http://localhost:8080`.

### 4. Test with curl (simulating auth headers)

Since the application expects authentication headers, you need to provide them:

#### For Azure Easy Auth mode (default):

```bash
# View SSH keys page
curl -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     http://localhost:8080/

# Add an SSH key
curl -X POST \
     -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     -d "publicKey=ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com" \
     http://localhost:8080/keys

# View DNS aliases page
curl -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     http://localhost:8080/dns

# Add a DNS alias
curl -X POST \
     -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     -d "source=gitea.example.com&destination=gitea.gitea.svc.cluster.local" \
     http://localhost:8080/dns
```

#### For oauth2-proxy mode:

```bash
export SSHBASTION_AUTH_MODE="oauth2_proxy"
./ssh-bastion web

# Then use different headers:
curl -H "X-Auth-Request-User: test-user-123" \
     -H "X-Auth-Request-Email: user@example.com" \
     http://localhost:8080/
```

### 5. Using a browser with a proxy

For browser testing, you can use a tool like [ModHeader](https://modheader.com/) browser extension to inject the required headers:

- `X-MS-CLIENT-PRINCIPAL-ID`: `test-user-123`
- `X-MS-CLIENT-PRINCIPAL-NAME`: `your-email@example.com`

Then navigate to `http://localhost:8080/`.

## Verifying Generated Files

### Check authorized_keys file

```bash
cat _tmp/data/authorized_keys/jump
```

This should contain all enabled SSH public keys.

### Check dnsmasq configuration

```bash
cat _tmp/data/dns/dnsmasq.d/generated.conf
```

This should contain `cname=` directives for all DNS aliases.

## Testing Without Auth Headers (401 Response)

```bash
curl -v http://localhost:8080/
# Should return: HTTP 401 Unauthorized
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test -v ./internal/keys
go test -v ./internal/dns
go test -v ./internal/storage
```

## Directory Structure After Use

```
_tmp/data/
├── users/
│   └── <user-uuid>/
│       └── keys/
│           ├── <fingerprint>.json
│           └── <fingerprint>.pub
├── authorized_keys/
│   └── jump
└── dns/
    ├── aliases.json
    └── dnsmasq.d/
        └── generated.conf
```

## Configuration Options

| Variable | Default (easy_auth) | Default (oauth2_proxy) | Description |
|----------|---------------------|------------------------|-------------|
| `SSHBASTION_DATA_DIR` | `/data` | `/data` | Root directory for file storage |
| `SSHBASTION_AUTH_MODE` | `easy_auth` | - | Authentication mode |
| `SSHBASTION_AUTH_USER_ID_HEADER` | `X-MS-CLIENT-PRINCIPAL-ID` | `X-Auth-Request-User` | Header containing user ID |
| `SSHBASTION_AUTH_EMAIL_HEADER` | `X-MS-CLIENT-PRINCIPAL-NAME` | `X-Auth-Request-Email` | Header containing email |
