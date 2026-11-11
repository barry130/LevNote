# Portable README wording correction: result

Status: corrected and frozen. Only `native/portable/README.md` changed; this
result is the only new handoff/evidence file.

Active task:
[`drawio-worker-portable-wording-task.md`](drawio-worker-portable-wording-task.md).

## Corrections

The README now uses these exact visible concepts:

- `重置全部引擎` in both reset descriptions;
- visible `阅读 / 编辑` or `分屏` modes in the draw.io source-mode instruction;
- `所选引擎的可执行程序、清单和必需的许可证/归属文件` for import, without
  claiming that an arbitrary complete directory is copied;
- whole-source warning ranges for draw.io, with page or cell scope stated in the
  warning message rather than implying precise node locations.

The README still contains exactly one `{{VERSION}}` placeholder. No version or
additional behavior claim was added.

Final README:

```text
0C18FEEF04A2EE146A231BCF767BDB0E57938DA902AA150D4CEFC17AB58C02E5  11411  native/portable/README.md
```

The earlier prepared README hash was
`3A1CE0C55572339BA857925E27D69F1B93C69090F93E8FAE550F96D96D1E52BC`.

## Preserved evidence

The original preparation result and its link-check evidence were not rewritten:

```text
F90011BDC3417970A07827463AB32B38B30F88DE3A0F8FF214D6E1DCABD7B17B  3961  native/handoffs/drawio-worker-portable-docs-result.md
C96EA8AB83B686EE3BFC8A2C8E86EEFF308B1B9DB6194A1EFE595CEAB62D8E69   395  native/handoffs/drawio-worker-portable-docs-evidence/evidence-manifest.sha256
```

No link rescan, test, build, package, launch, UI action, worker invocation, or
resource measurement ran. Final Stage2 tests remain pending, so this result
claims only the wording corrections above.

The README and this result are frozen for root reconciliation with the final
Stage2 result. `.18` remains immutable, and there is no `.19` package or
acceptance claim.
