# SARIF Integration

SARIF output lets GitHub show Kube ShipGuard findings inline in code scanning views.

```bash
kube-shipguard scan deploy --format sarif --output kube-shipguard.sarif
```

Upload the file with `github/codeql-action/upload-sarif`:

```yaml
- uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: kube-shipguard.sarif
```

Use `--fail-on high` when the pipeline should block only on severe release risks.
