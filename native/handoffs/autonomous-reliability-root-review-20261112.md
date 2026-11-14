# Autonomous reliability: Root source-stage review

Date: 2026-11-12. Reviewer: DSH lead. This is a read-only review of the named working-tree source, not a completed task-5, whole-application or platform acceptance.

## Identity and review boundary

- Development comparison baseline resolved to `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`; current committed main resolved to `8d4f43239dc30bfeb083dafeec7a02ed0f2da2d3`.
- Reviewed the working-tree diff against that baseline for the four tracked files below, and read the complete newly added autonomous test separately. These changes are not yet committed. A committed `base...HEAD` review must not be represented as including this uncommitted test or source.
- Root independently compared the size and full SHA256 of all five host mirrors to the box-origin freeze. Every comparison returned true. No Go, gofmt, test, vet, build, product launch or semantic source modification occurred on the host.
- Source-reading standards: [native coordination](<F:/AIproject/LevNote/native/AGENTS.md>), the applicable architecture/error/byte-fidelity rules in the [root handbook](<F:/AIproject/LevNote/AGENTS.md>) and [contribution guide](<F:/AIproject/LevNote/CONTRIBUTING.md>), and the [finite delivery scope](<F:/AIproject/LevNote/native/handoffs/autonomous-native-delivery-plan-20261112.md>). The native-specific remote-only policy overrides legacy pnpm/Tauri execution instructions.

| Frozen source | Bytes | SHA256 |
|---|---:|---|
| [reopen_closed.go](<F:/AIproject/LevNote/native/app/reopen_closed.go>) | 8564 | `a6b8013854d4c27d68e2ae1153525b0764d39b8892a700ce8804b206e485da44` |
| [reference.go](<F:/AIproject/LevNote/native/app/reference.go>) | 8049 | `9c5ebfd54b1bc1ed1de659bef13d34b29018a0cc2cfd572af91598e7e57f98cd` |
| [reference_test.go](<F:/AIproject/LevNote/native/app/reference_test.go>) | 7133 | `50451754b6389d8642c78295df32876f5b21ca103bd60c07d9117f3c7acabb4a` |
| [binary_actions.go](<F:/AIproject/LevNote/native/app/binary_actions.go>) | 10163 | `8174e9a8b3b89fd22a349292fd8d7c39cb905d655400fe7ba581255ffef8111b` |
| [autonomous_reliability_test.go](<F:/AIproject/LevNote/native/app/autonomous_reliability_test.go>) | 9467 | `2a3c83ffb229270aec79840aa072da13507f260e5d37261d65f5c92c20f5ce33` |

## Source-stage observations

1. [Native absolute-path admission](<F:/AIproject/LevNote/native/app/reopen_closed.go#L38-L67>) now uses the actual host's absolute-path predicate, rejects doubled or mixed-separator roots before cleaning, and retains complete drive-absolute Windows admission. The UTF-8/NUL/128-KiB checks apply both before and after canonicalization. Closed history still contains only at most 16 path/ID records; no document, draft, buffer or view is retained.
2. [Canonical comparison](<F:/AIproject/LevNote/native/app/reopen_closed.go#L69-L91>) preserves Windows folded comparison while avoiding false merging of two real Unix files differing only by case. The same comparison is used by the existing duplicate-open and [binary duplicate route](<F:/AIproject/LevNote/native/app/binary_actions.go#L216-L270>); the binary diff changes only that comparator and removes its now-unused import.
3. [Reference classification](<F:/AIproject/LevNote/native/app/reference.go#L34-L95>) checks doubled roots before accepting a native absolute path. Complete Windows drive literals keep Windows lexical cleaning on non-Windows test hosts. Relative references still resolve from the saved source document's directory, never implicitly from the process cwd. External schemes remain visibly refused.
4. [File URL handling](<F:/AIproject/LevNote/native/app/reference.go#L131-L175>) uses the already-decoded URL Path, or one decode for Opaque. It removes only the extra `/C:/...` URL slash, preserving a genuine Unix root. Host/user/query/ForceQuery/fragment/NUL and encoded doubled-root refusal remain. Rejecting a literal `#` also catches an empty fragment delimiter, while `%23` remains a literal filename character.
5. The new [five autonomous cases](<F:/AIproject/LevNote/native/app/autonomous_reliability_test.go>) use public App open/close flows, public editor Host reference entry and public Tester key routing. Expected text and disk bytes are independent literals. They cover current clean BOM/CRLF reopen, native case identity, native absolute/file URL references, bare-fragment/query/NUL refusal versus an encoded hash filename, and a missing-file failure followed by a real disk restoration and successful retry. No closed-history slice, private selection or receipt is injected.
6. The original [Windows uppercase close-history assertions](<F:/AIproject/LevNote/native/app/reopen_closed_test.go#L266-L283>) and [running-image cancellation oracle](<F:/AIproject/LevNote/native/app/reopen_closed_test.go#L225-L263>) remain in the existing test file. The original line-234 panic was the empty `closedFiles` history index, not a nil Tabs index. No new slice guard or test skip disguises that symptom.

## Standards and spec status

The reviewed source-stage changes contain no identified blocking documented-standard breach or demonstrated contradiction of the finite native reliability scope. Helpers remain local classification/identity functions rather than new platform, network or document-editor abstractions. The existing open/adoption/Flush/read-only/ordered-navigation barriers were not replaced.

This is Root's manual source-stage assessment. It is not a completed parallel Standards/Spec review of a committed source group, a compilation result, an affected-suite result, a measured resource improvement, or a Windows acceptance.

## Evidence and remaining gates

- The first reopen RED/GREEN receipts were independently read in the preceding stage; they prove that selected public flow only. The later `native-path-focused-01` owner report records exit 0, ten top-level and 29 nested selected cases. Root's full source/byte review does not enlarge that run's selection into full coverage.
- Complete affected case/reference/reopen/binary regression, actual vet disposition, source-before/after/snapshot equality and owned-process cleanup still require final primary-receipt review. Preserve all earlier compile/behavior failures.
- Existing Windows-specific assertions must stay intact. A concrete Windows-only expectation failing under Linux is reported separately; it must not be skipped, weakened or used to claim Windows passed.
- The validation-helper started-metadata failure has an independently exercised RED and owner-reported GREEN/normal-control correction in a separately created helper03. Those are helper lifecycle checks with `product_executed=false`, not Go/product PASS. Their source and raw receipts are a separate review item.
- All product commands remain serialized under native-reliability on the already authorized box. No consumed original run or label is relaunched. No application package is accepted here; `.29` remains the latest accepted runnable.

Next executable step: read the new helper03's owned-child cleanup code and original RED/GREEN receipts, then collect the fresh draw.io incident and experience revision03 Tree/25 results from the sole execution owner before task-5 affected-suite closure and main integration.
