# Kube ShipGuard

[![CI](https://github.com/yankawai/kube-shipguard/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/yankawai/kube-shipguard/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![SARIF](https://img.shields.io/badge/SARIF-enabled-2E3440)](.github/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Kube ShipGuard is an open-source Kubernetes release-readiness scanner for platform and DevOps teams. It checks rendered manifests before deployment and blocks risky workloads in CI.

The scanner focuses on production signals that reviewers usually look for manually: probes, resources, non-root containers, read-only filesystems, image tags, PodDisruptionBudgets, NetworkPolicies, and risky Secret/ConfigMap patterns.

## Why it exists

Kubernetes manifests often pass syntax validation while still being unsafe to ship. Kube ShipGuard adds a release gate that answers a more useful question:

> Is this workload ready to survive production traffic, disruption, and security review?

## Features

- Scans plain Kubernetes YAML and multi-document files.
- Supports `Deployment`, `StatefulSet`, `DaemonSet`, `Pod`, `Service`, `Secret`, `ConfigMap`, `PodDisruptionBudget`, and `NetworkPolicy`.
- Detects missing probes, missing resource requests/limits, unsafe security contexts, mutable image tags, single-replica workloads, missing PDBs, and missing NetworkPolicies.
- Emits text, JSON, and SARIF output.
- Includes an interactive terminal review mode for local manifest reviews.
- Supports CI failure gates by severity.
- Ships with a reusable GitHub Action.

## Quick start

```bash
go run ./cmd/kube-shipguard scan examples/secure --format text --fail-on high
```

Build the binary:

```bash
make build
./bin/kube-shipguard scan examples/secure
```

Generate SARIF for GitHub code scanning:

```bash
go run ./cmd/kube-shipguard scan deploy --format sarif --output kube-shipguard.sarif
```

Run the unsafe example without failing the process:

```bash
make demo
```

Open an interactive terminal review:

```bash
go run ./cmd/kube-shipguard review examples/unsafe
```

## Example output

```text
Kube ShipGuard found 7 findings

HIGH  KSG004 default/api Deployment/api container api allows privilege escalation
HIGH  KSG006 default/api Deployment/api container api uses mutable image tag latest
MED   KSG001 default/api Deployment/api container api has no readiness probe
```

## Exit behavior

`--fail-on` controls the minimum severity that returns a non-zero exit code:

- `none` never fails;
- `low` fails on low, medium, or high;
- `medium` fails on medium or high;
- `high` fails only on high findings.

## GitHub Action

```yaml
name: kube-shipguard

on:
  pull_request:

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: yankawai/kube-shipguard@v1
        with:
          path: deploy
          fail-on: high
          format: sarif
          output: kube-shipguard.sarif
```

## Checks

| Rule | Severity | Description |
| --- | --- | --- |
| KSG001 | Medium | Container has no readiness probe |
| KSG002 | Medium | Container has no liveness probe |
| KSG003 | Medium | Container has incomplete CPU/memory requests or limits |
| KSG004 | High | Container allows privilege escalation |
| KSG005 | Medium | Container root filesystem is writable |
| KSG006 | High | Container uses a mutable image tag |
| KSG007 | High | Workload can run as root |
| KSG008 | Medium | Container does not drop Linux capabilities |
| KSG009 | Medium | Replicated workload has fewer than two replicas |
| KSG010 | Medium | Workload has no matching PodDisruptionBudget |
| KSG011 | Medium | Workload has no matching NetworkPolicy |
| KSG012 | High | Secret manifest is stored in repository YAML |
| KSG013 | High | ConfigMap key looks like a secret |
| KSG014 | Low | Service exposes LoadBalancer directly |

## Documentation

- [Configuration](docs/configuration.md)
- [SARIF integration](docs/sarif.md)
- [Security policy](SECURITY.md)
