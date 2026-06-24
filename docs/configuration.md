# Configuration

Kube ShipGuard is intentionally CLI-first. The current configuration surface is provided through flags:

```bash
kube-shipguard scan ./deploy \
  --format sarif \
  --output kube-shipguard.sarif \
  --fail-on high
```

## Flags

- `--format`: `text`, `json`, or `sarif`.
- `--output`: file path for JSON or SARIF output. Text output defaults to stdout.
- `--fail-on`: `none`, `low`, `medium`, or `high`.

## Input

The scanner accepts files and directories. Directories are walked recursively. Files ending in `.yaml` or `.yml` are parsed as Kubernetes manifests.
