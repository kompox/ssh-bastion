---
id: task-20260104i-k8s-manifests-helm
title: Kubernetes: manifests / Helm
titleJa: Kubernetes: manifests / Helm
status: in-progress
updated: 2026-01-05T21:20:01Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Kubernetes: manifests / Helm

## Background

The project currently provides container images and local dev/test workflows, but does not yet include Kubernetes manifests (or a Helm chart) for deployment.

This work is tracked in [design-roadmap].

## Goal

Add initial Kubernetes deployment artifacts (Helm chart) for running ssh-bastion.

Publish chart name: `kompox-ssh-bastion` (avoid collisions with existing `ssh-bastion` charts).

## Requirements

- Keep scope minimal and runnable
- Persist data under `/data` via PVC
- Expose SSH as LoadBalancer and web as cluster-internal
- Use an in-Pod edge proxy for TLS termination:
  - edge proxy: Traefik
  - support Let's Encrypt
  - support Azure Key Vault TLS certificates via Secrets Store CSI driver mount
- Use an in-Pod auth proxy for OIDC (choose one):
  - oauth2-proxy
  - Azure Easy Auth middleware: `mcr.microsoft.com/appsvc/middleware:stage6`
- Support TLS cert sources:
  - Let's Encrypt (typically via cert-manager Secret)
  - Azure Key Vault certificate via Secrets Store CSI driver mount
- Include health checks/readiness
- Document required capabilities (e.g., binding to ports 22/53)

## Non-goals

- Production-grade Helm chart features (values matrix, advanced RBAC, etc.)
- Multi-tenant cluster considerations

## Plan & Checklist

- [x] Confirm Helm chart (no plain manifests)
- [x] Initialize chart skeleton at `deploy/helm/kompox-ssh-bastion/` (start with `Chart.yaml`)
- [x] Define Pod topology: `sshd`, `ssh-bastion`, `traefik`, auth proxy
- [x] Define auth proxy: oauth2-proxy `quay.io/oauth2-proxy/oauth2-proxy:latest`
- [x] Define auth proxy: Azure Easy Auth `mcr.microsoft.com/appsvc/middleware:stage6`
- [x] Add design doc for auth proxy containers (`design-auth-proxy`)
- [ ] Define Traefik TLS sources and configuration (Let's Encrypt + Azure Key Vault CSI)
- [x] Define `values.yaml` switches:
  - `traefik.tls.mode`: `letsEncrypt` | `secret`
  - `authProxy.provider`: `oauth2Proxy` | `azureEasyAuth`
- [ ] Add Service(s) and IngressRoute/Ingress (web)
- [ ] Add probes and securityContext
- [ ] Add minimal documentation (how to apply)

## Progress

- 2026-01-04T22:17:42Z
  - Task created and moved to IN-PROGRESS in roadmap

- 2026-01-04T22:24:14Z
  - Clarified requirements: OIDC auth proxy + TLS termination, and TLS cert sources (Let's Encrypt + Azure Key Vault CSI)

- 2026-01-04T22:31:07Z
  - Updated target Helm chart topology: Traefik edge proxy (Let's Encrypt + Azure Key Vault CSI) and auth proxy choice (oauth2-proxy vs Azure Easy Auth middleware)

- 2026-01-04T22:59:11Z
  - Decided chart publication name: `kompox-ssh-bastion`; plan to create `charts/kompox-ssh-bastion/Chart.yaml` first

- 2026-01-05T11:52:10Z
  - Updated repo layout so the Helm chart lives at `deploy/helm/kompox-ssh-bastion/` (moved from `charts/kompox-ssh-bastion/`) and updated docs/scripts accordingly

- 2026-01-05T19:15:19Z
  - Added [design-auth-proxy] to document auth proxy container configuration (oauth2-proxy and Azure Easy Auth) for Helm and Compose

- 2026-01-05T20:16:01Z
  - Fixed Azure Easy Auth middleware startup by passing required Host.* settings via command-line args (e.g. `/Host.DestinationHostUrl=...`) instead of env vars

- 2026-01-05T20:41:52Z
  - Added a per-auth-mode "Sign Out" link in the web UI header (`easy_auth` and `oauth2_proxy`), and updated templates to receive `AuthMode`

- 2026-01-05T20:53:03Z
  - Changed ssh-bastion defaults for `oauth2_proxy` mode to read identity from `X-Forwarded-User` / `X-Forwarded-Email` and aligned design docs

- 2026-01-05T21:20:01Z
  - Updated [design-k8s-helm] Quickstart to include both `oauth2_proxy` and `easy_auth` sections, and always use Let’s Encrypt staging `caServer`

## References

- [design-roadmap] - Development Roadmap
- [design-containers] - Containers (image + runtime topology)
- [design-auth-proxy] - Auth proxy containers (oauth2-proxy / Azure Easy Auth)

[design-roadmap]: ../design/design-roadmap.md
[design-containers]: ../design/design-containers.md
[design-auth-proxy]: ../design/design-auth-proxy.md