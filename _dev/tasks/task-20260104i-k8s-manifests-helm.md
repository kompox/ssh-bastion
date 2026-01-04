---
id: task-20260104i-k8s-manifests-helm
title: Kubernetes: manifests / Helm
titleJa: Kubernetes: manifests / Helm
status: in-progress
updated: 2026-01-04T22:59:11Z
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

- [ ] 1) Confirm Helm chart (no plain manifests)
- [ ] 2) Initialize chart skeleton at `charts/kompox-ssh-bastion/` (start with `Chart.yaml`)
- [ ] 3) Define Pod topology: `sshd`, `ssh-bastion`, `traefik`, auth proxy
- [ ] 4) Decide auth proxy: oauth2-proxy vs `mcr.microsoft.com/appsvc/middleware:stage6`
- [ ] 5) Define Traefik TLS sources and configuration (Let's Encrypt + Azure Key Vault CSI)
- [ ] 6) Define `values.yaml` switches:
  - `traefik.tls.mode`: `letsEncrypt` | `secret`
  - `authProxy.provider`: `oauth2Proxy` | `azureEasyAuth`
- [ ] 7) Add Service(s) and IngressRoute/Ingress (web)
- [ ] 8) Add probes and securityContext
- [ ] 9) Add minimal documentation (how to apply)

## Progress

- 2026-01-04T22:17:42Z
  - Task created and moved to IN-PROGRESS in roadmap

- 2026-01-04T22:24:14Z
  - Clarified requirements: OIDC auth proxy + TLS termination, and TLS cert sources (Let's Encrypt + Azure Key Vault CSI)

- 2026-01-04T22:31:07Z
  - Updated target Helm chart topology: Traefik edge proxy (Let's Encrypt + Azure Key Vault CSI) and auth proxy choice (oauth2-proxy vs Azure Easy Auth middleware)

- 2026-01-04T22:59:11Z
  - Decided chart publication name: `kompox-ssh-bastion`; plan to create `charts/kompox-ssh-bastion/Chart.yaml` first

## References

- [design-roadmap] - Development Roadmap
- [design-containers] - Containers (image + runtime topology)

[design-roadmap]: ../design/design-roadmap.md
[design-containers]: ../design/design-containers.md
