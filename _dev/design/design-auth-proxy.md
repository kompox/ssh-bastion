---
id: design-auth-proxy
title: Auth proxy containers (oauth2-proxy / Azure Easy Auth)
status: draft
updated: 2026-01-05T20:43:36Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Design: Auth proxy containers (oauth2-proxy / Azure Easy Auth)

This doc describes how ssh-bastion is deployed behind an auth proxy container.

Scope:

- oauth2-proxy (OIDC)
- Azure Easy Auth middleware container (`mcr.microsoft.com/appsvc/middleware:stage6`)
- Kubernetes (Helm) and Docker Compose

This design complements:

- [design-overview] (product architecture and behavior)
- [design-k8s-helm] (Kubernetes packaging and values contract)
- [design-containers] (container image contents and runtime assumptions)

## Goals

- Define a clear contract between auth proxy and ssh-bastion.
- Make configuration explicit and reproducible across Kubernetes and Compose.
- Keep authentication and routing behavior consistent between environments.

## Non-goals

- Choosing a single identity provider (IdP).
- Building a general-purpose auth proxy framework.
- Full production hardening guidance for every cluster.

## Terminology

- **Auth proxy**: the container that enforces user authentication (OIDC or platform auth).
- **Upstream**: ssh-bastion web UI service behind the auth proxy.
- **Edge proxy**: TLS termination and routing proxy (e.g., Traefik).

## Contract (auth proxy → ssh-bastion)

Document what ssh-bastion expects from the proxy, and what the proxy must guarantee.

- **Authenticated identity headers**:
  - oauth2-proxy: `X-Forwarded-User` and `X-Forwarded-Email` (enabled by oauth2-proxy `--pass-user-headers=true`)
  - Azure Easy Auth middleware: `X-MS-CLIENT-PRINCIPAL-ID` and `X-MS-CLIENT-PRINCIPAL-NAME`
- **Auth mode configuration**:
  - `SSHBASTION_AUTH_MODE=oauth2_proxy` when deployed behind oauth2-proxy
  - `SSHBASTION_AUTH_MODE=easy_auth` when deployed behind Azure Easy Auth middleware
- **Request routing**:
  - `/oauth2/*` (oauth2-proxy endpoints) routing rules (K8s/Compose)
  - `/*` to ssh-bastion (protected)

## oauth2-proxy (OIDC)

### Container configuration

- Provider: OIDC
- OIDC issuer URL
- Client ID / client secret
- Cookie secret
- Redirect URL (must match the public host)

oauth2-proxy listens on `:4180` and proxies to ssh-bastion at `http://127.0.0.1:8080/`.

Paths reserved by oauth2-proxy:

- `/oauth2/*` (login/callback/sign-out endpoints)

Sign-out endpoint (used by the ssh-bastion UI header link in `oauth2_proxy` mode):

- `/oauth2/sign_out?rd=/`

### Kubernetes (Helm)

Traefik routing (current chart behavior):

- `Host(<host>) && PathPrefix(/oauth2)` → oauth2-proxy (`http://127.0.0.1:4180`)
- `Host(<host>) && PathPrefix(/)` → oauth2-proxy (`http://127.0.0.1:4180`)

oauth2-proxy container settings (current chart behavior):

- `--provider=oidc`
- `--http-address=0.0.0.0:4180`
- `--upstream=http://127.0.0.1:8080/`
- `--reverse-proxy=true`
- `--pass-user-headers=true` (emits `X-Forwarded-User` / `X-Forwarded-Email`)
- `--oidc-issuer-url=<issuerURL>`
- `--client-id=<clientID>`
- `--redirect-url=https://<host>/oauth2/callback`
- `--email-domain=*` (or configured domains)

Secrets (via Kubernetes Secret refs):

- OIDC client secret → `OAUTH2_PROXY_CLIENT_SECRET`
- Cookie secret → `OAUTH2_PROXY_COOKIE_SECRET`

### Compose

The same topology as Helm applies:

- edge proxy terminates TLS and routes both `/oauth2/*` and `/*` to oauth2-proxy
- oauth2-proxy proxies to ssh-bastion (upstream)

Minimum required configuration is the same as the container configuration above.

## Azure Easy Auth middleware

### Container configuration

Container Image: `mcr.microsoft.com/appsvc/middleware:stage6`

Azure Easy Auth in SSH Bastion supports the following configuration:

- Single repilica deployment
- Persistent token store on filesystem at `/data` (RWO mount)
- Microsoft Entra ID (`azureActiveDirectory`) as Identity Provider
- Route reserved by Azure Easy Auth middleware: `/.auth/*`
- LoadBalancer: 443 (HTTPS), 22 (SSH)
  - -> traefik (0.0.0.0:443) -> easyauth (127.0.0.1:8000) -> ssh-bastion (127.0.0.1:8080)
  - -> sshd (0.0.0.0:22) -> ssh-bastion (127.0.0.1:53) 

Host settings (pass as command-line args):

Notes:

- The middleware expects dotted keys (e.g. `Host.DestinationHostUrl`).
- In Kubernetes, env var names cannot contain `.`, and `Host__DestinationHostUrl`
  does not map to `Host.DestinationHostUrl` for this container.

```
/Host.ListenUrl=http://127.0.0.1:8000
/Host.DestinationHostUrl=http://127.0.0.1:8080 # ssh-bastion
/Host.AutoHealingMiddlewareEnabled=false
/Host.UseConsoleLogging=true
/Host.UseFileLogging=true
/Host.RewriteHostHeader=false
```

Variables (via K8s Secret / ConfigMap environment variables):

```
WEBSITE_AUTH_FROM_FILE=true
WEBSITE_AUTH_FILE_PATH=/app/easyauth-config.json
```

Secrets (via K8s Secret environment variables):

```
MICROSOFT_PROVIDER_AUTHENTICATION_SECRET=<CLIENT-SECRET>
WEBSITE_AUTH_ENCRYPTION_KEY=$(openssl rand -hex 32)
WEBSITE_AUTH_SIGNING_KEY=$(openssl rand -hex 32)
```

Configuration file `/app/easyauth-config.json` (via a mounted volume or ConfigMap/Secret):

```json
{
  "platform": {
    "enabled": true
  },
  "globalValidation": {
    "requireAuthentication": true,
    "unauthenticatedClientAction": "RedirectToLoginPage",
    "redirectToProvider": "azureActiveDirectory",
    "excludedPaths": []
  },
  "identityProviders": {
    "azureActiveDirectory": {
      "enabled": true,
      "registration": {
        "openIdIssuer": "https://sts.windows.net/<TENANT-ID>/v2.0",
        "clientId": "<CLIENT-ID>",
        "clientSecretSettingName": "MICROSOFT_PROVIDER_AUTHENTICATION_SECRET"
      },
      "login": {
        "disableWWWAuthenticate": false,
        "loginParameters": [
          "scope=openid profile email offline_access"
        ]
      },
      "validation": {
        "allowedAudiences": [
          "<CLIENT-ID>"
        ],
        "defaultAuthorizationPolicy": {
          "allowedPrincipals": {
            "groups": ["<GROUP-ID-1>","<GROUP-ID-2>","..."]
          }
        }
      },
      "isAutoProvisioned": false
    }
  },
  "login": {
    "tokenStore": {
      "enabled": true,
      "fileSystem": {
        "directory": "/data"
      }
    },
    "preserveUrlFragmentsForLogins": false
  },
  "httpSettings": {
    "requireHttps": false,
    "forwardProxy": {
      "convention": "Standard"
    }
  }
}
```

HTTP headers sent to upstream (ssh-bastion):

```
X-MS-CLIENT-PRINCIPAL: (base64-encoded JSON blob, currently ignored by ssh-bastion)
X-MS-CLIENT-PRINCIPAL-ID: <USER-ID> (captured via SSHBASTION_AUTH_USER_ID_HEADER)
X-MS-CLIENT-PRINCIPAL-NAME: <USER-EMAIL> (captured via SSHBASTION_AUTH_EMAIL_HEADER)
X-MS-CLIENT-PRINCIPAL-IDP: aad
```

Authentication HTML snippets:

```html
<a href="/.auth/logout?post_logout_redirect_uri=/">Sign Out</a>
```

References:

- https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-customize-sign-in-out
- https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-user-identities
- https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-oauth-tokens
- https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-file-based

### Kubernetes (Helm)

This section enumerates the Kubernetes resources (and keys) that the Helm chart should create or reference.

#### Secrets

Create (or reference) this Secret:

- Secret name: `ssh-bastion-easyauth-secrets`
  - `MICROSOFT_PROVIDER_AUTHENTICATION_SECRET`: Entra ID application client secret
  - `WEBSITE_AUTH_ENCRYPTION_KEY`: 32-byte hex string (suggested generation: `openssl rand -hex 32`)
  - `WEBSITE_AUTH_SIGNING_KEY`: 32-byte hex string (suggested generation: `openssl rand -hex 32`)

Notes:

- These keys must be generated beforehand and stable across upgrades if you want sessions/tokens to survive Pod restarts.

#### ConfigMaps

Create (or reference) these ConfigMaps:

- ConfigMap name: `ssh-bastion-easyauth-config`
  - `easyauth-config.json`: JSON content mounted to `/app/easyauth-config.json`

Notes:

- The middleware expects dotted keys (e.g., `Host.DestinationHostUrl`). For Kubernetes/Helm, pass these Host settings as container args (e.g., `/Host.DestinationHostUrl=...`) rather than env vars.

#### Mounts

- Mount `/data` as a persistent volume (RWO) for the Easy Auth token store.

### Compose

- Service wiring and env vars
- Local testing constraints (what can/can’t be simulated)

## Security considerations

- Ensure ssh-bastion does not accept unauthenticated requests directly.
- Clarify trust boundary: only accept identity headers from the trusted proxy.
- Avoid leaking tokens/secrets via logs.

## Open questions

- Exact header contract for both proxies (names, normalization, required fields).
- Whether both proxy modes must be supported long-term.
- How logout/session invalidation should behave.

## References

- [design-overview] - High-level architecture and behavior
- [design-k8s-helm] - Kubernetes packaging (Helm)
- [design-containers] - Containers (image + runtime topology)

[design-overview]: ../design/design-overview.md
[design-k8s-helm]: ../design/design-k8s-helm.md
[design-containers]: ../design/design-containers.md
