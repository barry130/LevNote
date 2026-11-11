# Closed-file reopen source slice accepted

Accepted2026-11-10. Root reviewed the implementation/result, actual final
source identities, focused execution logs and the independent static review.
The prior context's760x480 capture review is inherited without repeating it.
Implementation/test/build work was delegated; root did not run product checks.

Six runtime files and one test total124304bytes. Frozen manifest
reopen-closed-file-evidence-20261110/source-frozen.json SHA256
5875F6B4BC504348475E71427DD25CE56ED89C255177DF219FF6FD3D486F464E.
Implementation result SHA256
D60CCED82651F28421E126675D2BF1193ED3EF005F536BF75E96D009A2EA2FAB.
Independent static review SHA256
3A7A395E35FBCB25B595A35C93ECE756F79EF0FEEE3657B0D1848989F273714D
matches all six runtime files; the final additional test is separately recorded.

Accepted Ctrl+Shift+T/native menu,16 bounded path/ID-only entries, successful
individual-close recording, current-disk reopening, guarded actual adoption
receipts, no discarded edit retention, exact stale-ID rejection and explicit
failure/retry. Failed Flush, utility preedit, modal/recovery/shutdown barriers
preserve existing ownership. Disabled menu KeepFocus fixes the observed loss
of utility-field focus; whole-window close cancels pending reopen without
recording all tabs. No core/session/decoder/vendor changes.

Focused03 PASS3.144s; final close/shutdown/capture05 PASS2.142s; final menu-ID/
held-key06 PASS1.648s, all exit0 with actual logs.18 top-level groups across
these runs. Initial failed01 remains recorded. No full app suite/vet/package
yet; this is source-slice acceptance, not a new runnable release or physical
Windows input verification.

Astra releases app implementation ownership. Live-tab admission now proceeds
sequentially under live-tab-admission-implementation-task.md, preserving these
receipt/ownership semantics. One combined final app test/vet follows both
increments. Current accepted runnable remains .26 and immutable.
