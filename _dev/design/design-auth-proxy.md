---
id: design-auth-proxy
title: Auth proxy containers (oauth2-proxy / Azure Easy Auth)
status: draft
updated: 2026-01-05T18:33:05Z
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

- **Authenticated identity headers**: (TBD; list exact header names used by ssh-bastion)
- **Auth mode configuration**:
  - `SSHBASTION_AUTH_MODE=oauth2_proxy` when deployed behind oauth2-proxy
  - (TBD) expected mode for Azure Easy Auth middleware
- **Request routing**:
  - `/oauth2/*` (oauth2-proxy endpoints) routing rules (K8s/Compose)
  - `/*` to ssh-bastion (protected)

## oauth2-proxy (OIDC)

### Container configuration

- OIDC issuer URL
- Client ID / client secret
- Cookie secret
- Redirect URL and callback behavior

### Kubernetes (Helm)

- Where secrets live (Kubernetes Secrets)
- How values map to oauth2-proxy args/envs
- How Traefik (or another edge proxy) routes:
  - forward-auth / middleware configuration (TBD)
  - `/oauth2/*` path routing (TBD)

### Compose

- Environment variables / volumes
- How the reverse proxy routes to oauth2-proxy and ssh-bastion
- Local testing notes (ports, callbacks)

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

Variables (via K8s ConfigMap environment variables):

```
Host.ListenUrl=http://127.0.0.1:8000
Host.DestinationHostUrl=http://127.0.0.1:8080 # ssh-bastion
Host.AutoHealingMiddlewareEnabled=false
Host.UseConsoleLogging=true
Host.UseFileLogging=true
Host.RewriteHostHeader=false
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
<a href="/.auth/login/aad?post_login_redirect_uri=/home">Sign in with Microsoft Entra</a>
<a href="/.auth/logout?post_logout_redirect_uri=/">Sign out</a>
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

- ConfigMap name: `ssh-bastion-easyauth-env`
  - `Host__ListenUrl`: `http://127.0.0.1:8000`
  - `Host__DestinationHostUrl`: `http://127.0.0.1:8080`
  - `Host__AutoHealingMiddlewareEnabled`: `false`
  - `Host__UseConsoleLogging`: `true`
  - `Host__UseFileLogging`: `true`
  - `Host__RewriteHostHeader`: `false`
  - `WEBSITE_AUTH_FROM_FILE`: `true`
  - `WEBSITE_AUTH_FILE_PATH`: `/app/easyauth-config.json`

- ConfigMap name: `ssh-bastion-easyauth-config`
  - `easyauth-config.json`: JSON content mounted to `/app/easyauth-config.json`

Notes:

- The middleware documentation uses dotted setting names (e.g., `Host.ListenUrl`). In Kubernetes, environment variable names cannot contain dots; this design standardizes on the double-underscore form (`Host__ListenUrl`) for Helm/Compose wiring.

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
