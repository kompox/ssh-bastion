---
id: design-k8s-helm
title: Kubernetes packaging (Helm)
status: draft
updated: 2026-01-05T21:17:35Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Kubernetes packaging (Helm)

This doc describes:

- the repository layout and conventions for Helm charts
- the intended responsibilities of the primary chart (`kompox-ssh-bastion`)
- the values contract (what is configurable, and what is intentionally fixed)

This design complements:

- [design-overview] (product architecture and behavior)
- [design-containers] (image contents + runtime topology assumptions)

## Goals

- Provide a repeatable Kubernetes deployment mechanism for this repo via Helm.
- Keep chart configuration minimal and explicit (avoid “magic defaults” that hide required inputs).
- Support the current MVP topology: single Pod, multiple containers, shared `/data` volume.
- Keep the chart aligned with the actual runtime flags/envs supported by the Go app.

## Non-goals

- Serving as a generic “bastion chart” for unrelated applications.
- Owning Ingress controller installation (Traefik/NGINX/etc.) at cluster scope.
- Defining a final production hardening profile (Pod Security, NetworkPolicies, etc.).
- Implementing Azure-specific integrations that require cluster-level prerequisites (these are tracked as follow-up work).

## Quickstart

This quickstart assumes:

- You already cloned this repository.
- You have access to a Kubernetes cluster (`kubectl` is configured).
- You have Helm installed.

You will need values for:

- public host name for the admin UI
- TLS (Let’s Encrypt email, if using TLS-ALPN-01)
- either oauth2-proxy OIDC settings, or Azure Easy Auth settings

### Common

1. **Pick a namespace and release name**

    Example:

    - Namespace: `ssh-bastion`
    - Release: `ssh-bastion`

2. **Create the namespace**

    ```bash
    NS=ssh-bastion
    RELEASE=ssh-bastion

    kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
    ```

3. **Pick TLS settings (staging)**

  In this doc, the examples always use Let’s Encrypt staging:

  - `edgeProxy.traefik.tls.letsEncrypt.caServer=https://acme-staging-v02.api.letsencrypt.org/directory`

### oauth2_proxy (oauth2-proxy)

1. **Create Kubernetes Secrets for oauth2-proxy**

    Create a cookie secret and OIDC client secret in the target namespace:

    ```bash
    kubectl -n "$NS" create secret generic oauth2-proxy-secrets \
      --from-literal=cookie-secret="$(openssl rand -base64 32)" \
      --from-literal=client-secret="REPLACE_ME" \
      --dry-run=client -o yaml | kubectl apply -f -
    ```

    Notes:

    - `issuerURL` should be the issuer base URL (not the `/.well-known/openid-configuration` URL).

2. **Install (or upgrade) the chart**

    From the repository root:

    ```bash
    helm upgrade --install "$RELEASE" deploy/helm/kompox-ssh-bastion \
      -n "$NS" \
      --create-namespace \
      --set host="bastion.example.com" \
      --set edgeProxy.traefik.tls.letsEncrypt.email="you@example.com" \
      --set edgeProxy.traefik.tls.letsEncrypt.caServer="https://acme-staging-v02.api.letsencrypt.org/directory" \
      --set authProxy.provider=oauth2Proxy \
      --set authProxy.oauth2Proxy.oidc.issuerURL="https://issuer.example.com/" \
      --set authProxy.oauth2Proxy.oidc.clientID="REPLACE_ME" \
      --set authProxy.oauth2Proxy.oidc.clientSecret.existingSecretName="oauth2-proxy-secrets" \
      --set authProxy.oauth2Proxy.cookieSecret.existingSecretName="oauth2-proxy-secrets"
    ```

### easy_auth (Azure Easy Auth middleware)

1. **Create Kubernetes Secrets for Easy Auth**

    Create the Easy Auth secret (client secret + encryption/signing keys) in the target namespace:

    ```bash
    kubectl -n "$NS" create secret generic ssh-bastion-easyauth-secrets \
      --from-literal=MICROSOFT_PROVIDER_AUTHENTICATION_SECRET="REPLACE_ME" \
      --from-literal=WEBSITE_AUTH_ENCRYPTION_KEY="$(openssl rand -hex 32)" \
      --from-literal=WEBSITE_AUTH_SIGNING_KEY="$(openssl rand -hex 32)" \
      --dry-run=client -o yaml | kubectl apply -f -
    ```

    Notes:

    - If you leave `authProxy.azureEasyAuth.config.existingConfigMapName` and `authProxy.azureEasyAuth.config.inlineJSON` empty, the chart generates `easyauth-config.json` from `authProxy.azureEasyAuth.identity.*`.

2. **Install (or upgrade) the chart**

    From the repository root:

    ```bash
    helm upgrade --install "$RELEASE" deploy/helm/kompox-ssh-bastion \
      -n "$NS" \
      --create-namespace \
      --set host="bastion.example.com" \
      --set edgeProxy.traefik.tls.letsEncrypt.email="you@example.com" \
      --set edgeProxy.traefik.tls.letsEncrypt.caServer="https://acme-staging-v02.api.letsencrypt.org/directory" \
      --set authProxy.provider=azureEasyAuth \
      --set authProxy.azureEasyAuth.identity.tenantID="REPLACE_ME" \
      --set authProxy.azureEasyAuth.identity.clientID="REPLACE_ME" \
      --set authProxy.azureEasyAuth.secrets.existingSecretName="ssh-bastion-easyauth-secrets"
    ```

### Verify

1. **Verify resources**

    ```bash
    kubectl -n "$NS" get pods
    kubectl -n "$NS" get svc
    ```

    If you don’t have external ingress/LB wiring yet, you can port-forward HTTP to confirm the app starts:

    ```bash
    kubectl -n "$NS" port-forward svc/${RELEASE}-kompox-ssh-bastion 8443:443
    ```

    Then open `https://localhost:8443/`.

## Repository layout

### Chart location

- Helm charts live under `deploy/helm/`.
- The primary chart for this repo is `deploy/helm/kompox-ssh-bastion/`.
- Rationale:
  - Avoid naming collisions with the repository name and other upstream charts.
  - Make the chart name stable for `helm install` / `helm upgrade` commands.

### Chart-level docs

- Each chart should have its own README (chart usage, values examples, prerequisites).
- Design docs (this doc) capture architectural intent and long-term compatibility rules.

## Scope and responsibilities

### What the chart owns

- Namespaced resources that deploy the application:
  - Deployment (single Pod with multiple containers)
  - Service(s) for exposing HTTP(S) and SSH
  - PersistentVolumeClaim (or an option to use an existing claim)
  - ConfigMaps/Secrets required for the chosen edge/auth components

### What the chart assumes exists

- A Kubernetes cluster with:
  - a StorageClass suitable for a writable `/data` volume
  - a DNS configuration that can resolve external names
- Optional prerequisites depend on features:
  - If using Let’s Encrypt TLS-ALPN-01: inbound TLS (TCP/443) must reach the edge proxy.
  - If integrating with an external identity provider: OIDC client credentials must exist.

## Configuration model (values)

The values contract is intentionally “small surface area”:

- Values should map to concrete behavior:
  - container args/flags
  - environment variables
  - Kubernetes resource properties
- Prefer explicit required values rather than silently inventing defaults.

### Host and routing

- One externally meaningful host name is assumed for the admin UI.
- The edge proxy is responsible for routing:
  - oauth2-proxy mode: route both `/oauth2/*` and `/*` to oauth2-proxy, which then proxies to ssh-bastion
  - Azure Easy Auth mode: route `/*` to the Easy Auth middleware, which then proxies to ssh-bastion (`/.auth/*` is reserved by the middleware)

### TLS modes

Support (at minimum) these conceptual modes:

- **Let’s Encrypt (TLS-ALPN-01)**: automatic certificate provisioning by the edge proxy.
- **Secret-provided**: use a pre-created TLS Secret.

(Concrete implementation details may vary per edge proxy choice.)

### Auth proxy modes

The chart should be able to express at least:

- **oauth2-proxy (OIDC)**
- **Azure Easy Auth middleware**

The chart sets the app auth mode and identity header mapping based on the selected auth proxy provider:

- oauth2-proxy: `SSHBASTION_AUTH_MODE=oauth2_proxy` and `X-Forwarded-User` / `X-Forwarded-Email`
- Azure Easy Auth: `SSHBASTION_AUTH_MODE=easy_auth` and `X-MS-CLIENT-PRINCIPAL-ID` / `X-MS-CLIENT-PRINCIPAL-NAME`

## Operational expectations

### Upgrades

- `helm upgrade` should be safe and predictable.
- Breaking changes to values should be avoided; if unavoidable, they must be called out in chart docs and release notes.

### Storage

- `/data` is the persistence root.
- The chart should support:
  - creating a PVC
  - referencing an existing PVC

### Observability

- All containers log to stdout/stderr.
- If metrics/logging sidecars are introduced later, they should be optional and documented.

## Testing strategy (chart)

Minimum expected checks:

- `helm lint` for template hygiene.
- `helm template` with a representative values set for the “happy path”.

Higher-level E2E tests are tracked elsewhere (see [design-e2e-testing]).

## Open questions / follow-up work

- TLS via Azure Key Vault + CSI (and how it composes with Let’s Encrypt usage).
- Alternative auth proxy choice(s) (e.g., Azure Easy Auth style front-end) and the exact header contract.
- Security hardening baseline (capabilities for binding to :53, read-only rootfs, PodSecurityContext).
- Scaling semantics (replicas > 1) given `/data` storage modes and auth/session behavior.

## References

- [design-overview] - High-level architecture and behavior
- [design-containers] - Container image and runtime topology
- [design-e2e-testing] - E2E testing design
- [design-roadmap] - Development roadmap

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-e2e-testing]: ../design/design-e2e-testing.md
[design-roadmap]: ../design/design-roadmap.md
