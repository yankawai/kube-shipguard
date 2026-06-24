# Contributing

Kube ShipGuard is designed for platform and DevOps teams that need practical CI gates for Kubernetes release readiness.

## Local checks

```bash
make validate
```

## Guidelines

- Keep rules deterministic and explainable.
- Prefer low false-positive rates over noisy policy coverage.
- Add tests for every new rule.
- Keep output stable enough for CI and SARIF integrations.
- Do not add rules that require cluster access.
