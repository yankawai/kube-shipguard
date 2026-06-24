# Configuration

Kube ShipGuard is CLI-first, but production workflows usually need four layers:

- input selection;
- accepted suppressions;
- legacy baselines;
- CI failure thresholds.

## Scan flags

```bash
kube-shipguard scan ./deploy \
  --format sarif \
  --output kube-shipguard.sarif \
  --fail-on high
```

- `--format`: `text`, `json`, or `sarif`.
- `--output`: file path for JSON or SARIF output. Text output defaults to stdout.
- `--fail-on`: `none`, `low`, `medium`, or `high`.
- `--config`: config file with expiring suppressions.
- `--baseline`: baseline file used to ignore known findings.
- `--changed-from`: git ref or range used to scan only changed YAML files.

## Input

The scanner accepts files and directories. Directories are walked recursively. Files ending in `.yaml` or `.yml` are parsed as Kubernetes manifests.

```bash
kube-shipguard scan deploy service.yaml
```

## Diff-aware PR scanning

Use `--changed-from` when a repository already has legacy risk and the pull request should be judged by what it changes.

```bash
kube-shipguard scan deploy --changed-from origin/main --fail-on high
```

In GitHub Actions, set `fetch-depth: 0` on `actions/checkout` when the base ref may not be present in the shallow checkout.

If no path is provided, Kube ShipGuard uses `.` as the diff scope:

```bash
kube-shipguard scan --changed-from origin/main
```

## Baselines

Baselines capture existing findings so CI can block only new release risk.

```bash
kube-shipguard baseline deploy --output .kube-shipguard-baseline.yaml
kube-shipguard scan deploy --baseline .kube-shipguard-baseline.yaml --fail-on high
```

Regenerate the baseline only after the team intentionally accepts the current risk state.

## Expiring suppressions

Suppressions are for reviewed exceptions. Each suppression must include a rule, reason, and expiry date.

```yaml
suppressions:
  - rule: KSG014
    kind: Service
    namespace: default
    name: api-public
    reason: public endpoint is reviewed and protected by external controls
    expires: 2026-12-31
```

Run with:

```bash
kube-shipguard scan deploy --config .kube-shipguard.yaml
```

Expired suppressions fail the scan configuration instead of silently hiding findings.

## Rendered manifests

Kube ShipGuard can scan rendered Helm and Kustomize output without committing rendered YAML.

```bash
kube-shipguard scan --helm-chart charts/api --helm-release api --helm-namespace prod --helm-values values-prod.yaml
kube-shipguard scan --kustomize overlays/prod
```

`--helm-values` can be repeated.
