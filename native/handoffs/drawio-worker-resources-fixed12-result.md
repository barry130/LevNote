# Fixed12 draw.io worker resource result

2026-11-10 Astra. One authorized matrix invocation, twelve actual requests,
12/12 PASS; wrapper exit0 and one summarizer invocation exit0. No retry,
warmup, discarded row, parallel request, host launch, build, Go test or
preflight rerun. Measurements and capture are frozen. Root was notified once
all twelve tester/worker lifecycles ended; other owners may now run tests.

PowerShell7.6.9 executable:
`C:\Users\canace\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\powershell\pwsh.exe`.
Working directory `F:\AIproject\LevNote`. Exact argument arrays are preserved
in the capture sibling's matrix-command.json and summary-command.json:

```text
-NoProfile -File F:\AIproject\LevNote\native\handoffs\drawio-worker-resources-matrix-wrapper-20261110\run-matrix.ps1 -OutputRoot F:\AIproject\LevNote\native\artifacts\acceptance\drawio-worker-resources-fixed12-20261110 -Activation RUN_FIXED_12_DRAWIO_WORKER_MATRIX
-NoProfile -File F:\AIproject\LevNote\native\handoffs\drawio-worker-resources-matrix-wrapper-20261110\summarize-matrix.ps1 -MatrixOutputRoot F:\AIproject\LevNote\native\artifacts\acceptance\drawio-worker-resources-fixed12-20261110
```

Wrapper SHA7947980b6bdc00fd5fe0171ca0b681726cbbbd55706f88286516c212433eae0e;
summary SHA1bd9455d41ba17c49b055dc4a30bbc72eeb387ce351278bf4af3b8178bc29929.
Wrapper capture start01:56:50.4951100Z; run01:56:51.2755444Z–01:57:03.9343822Z.
Windows amd64 build26200, Go1.27.3,16 logical CPUs/GOMAXPROCS16.

All raw rows below precede descriptive comparisons. Times are ns; CPU is the
Windows100ns counter unit, not nanoseconds. WS=sampled maximum working set;
private=sampled maximum private commit; OS WS/commit=observed OS peak counters.

| # | Fixture | Version | Ordinal | Export ns | WS bytes | Private bytes | OS WS bytes | OS commit bytes | Kernel100ns | User100ns | Lifetime ns | Reads |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | NormalMultiPage | .20 | 1 | 491305800 | 7872512 | 13332480 | 7872512 | 13332480 | 0 | 156250 | 154254100 | 7 |
| 2 | NormalMultiPage | .21 | 1 | 501547800 | 8032256 | 13713408 | 8032256 | 13713408 | 0 | 0 | 138183300 | 7 |
| 3 | NormalMultiPage | .21 | 2 | 467135500 | 7831552 | 13262848 | 7831552 | 13262848 | 0 | 156250 | 127588500 | 7 |
| 4 | NormalMultiPage | .20 | 2 | 499308000 | 7925760 | 13336576 | 7925760 | 13336576 | 0 | 0 | 139296600 | 8 |
| 5 | NormalMultiPage | .20 | 3 | 687122700 | 7925760 | 13328384 | 7925760 | 13328384 | 0 | 0 | 273837500 | 8 |
| 6 | NormalMultiPage | .21 | 3 | 577962000 | 7995392 | 13709312 | 7995392 | 13709312 | 156250 | 0 | 171596500 | 19 |
| 7 | MetadataHeavy | .21 | 1 | 656423700 | 8675328 | 14172160 | 8675328 | 14172160 | 156250 | 0 | 231851700 | 12 |
| 8 | MetadataHeavy | .20 | 1 | 735808700 | 8867840 | 14282752 | 8867840 | 14282752 | 156250 | 0 | 145982800 | 9 |
| 9 | MetadataHeavy | .20 | 2 | 565926300 | 8998912 | 14442496 | 8998912 | 14442496 | 0 | 156250 | 137452500 | 11 |
| 10 | MetadataHeavy | .21 | 2 | 586952700 | 8712192 | 14168064 | 8712192 | 14168064 | 156250 | 156250 | 145590900 | 9 |
| 11 | MetadataHeavy | .21 | 3 | 621045400 | 8896512 | 14602240 | 8896512 | 14602240 | 0 | 156250 | 190974500 | 9 |
| 12 | MetadataHeavy | .20 | 3 | 765697000 | 8982528 | 14401536 | 8982528 | 14401536 | 312500 | 0 | 156016500 | 17 |

Each cell below is median [minimum,maximum], n=3. OS peak WS and commit happened
to equal sampled maxima in each row here; both remain separately recorded.

| Fixture/version | Export ns | WS bytes | Private bytes | Lifetime ns |
|---|---|---|---|---|
| Normal .20 | 499308000 [491305800,687122700] | 7925760 [7872512,7925760] | 13332480 [13328384,13336576] | 154254100 [139296600,273837500] |
| Normal .21 | 501547800 [467135500,577962000] | 7995392 [7831552,8032256] | 13709312 [13262848,13713408] | 138183300 [127588500,171596500] |
| Metadata .20 | 735808700 [565926300,765697000] | 8982528 [8867840,8998912] | 14401536 [14282752,14442496] | 145982800 [137452500,156016500] |
| Metadata .21 | 621045400 [586952700,656423700] | 8712192 [8675328,8896512] | 14172160 [14168064,14602240] | 190974500 [145590900,231851700] |

| Fixture/version | Kernel100ns median [min,max] | User100ns median [min,max] |
|---|---|---|
| Normal .20 | 0 [0,0] | 0 [0,156250] |
| Normal .21 | 0 [0,156250] | 0 [0,156250] |
| Metadata .20 | 156250 [0,312500] | 0 [0,156250] |
| Metadata .21 | 156250 [0,156250] | 156250 [0,156250] |

No missing counters,123 reads/zero failed reads; interval2,000,000ns,
max20,000 attempts, actual7–19 per row. Twelve final memory/CPU reads passed;
all final OS lifetime-peak counters were retained and met the harness coverage
check. Zero limited/unknown coverage rows, zero limitation messages. This does
not make current-working-set samples continuous or reveal unobserved values.
CPU values are quantized (observed0/156250/312500units); zero is not zero work.
Export is managed request wall elapsed; process lifetime is creation-to-exit
wall time, separate from CPU and from the wrapper/tester lifetime.

Normal source28976B/SHAab822c28db64279a793810c45bbff7f6d7ecfd1cd44aaa1f59c48e00fedcaea2;
Metadata93639B/SHA504b19ffcf126353b79b34bcb5735f6a1cff21d5d515b23bc294342c7a61bcb1.
Every output598B/SHA3f990a4aac3209bafad94f44026bf14f3cec27ea44bdbd3169555947936472bf.
Each .20 row had exactly the two ordered expected canonical-container warnings;
each .21 row had none. Source hashes, selected-page metadata, installed engine
identity and one admission/verify/start/wait matched the accepted tester.

Twelve tester exits0, no timeout/process/cleanup/identity/artifact error. Each
worker exit0, WAIT_OBJECT_0, sampler joined, duplicate handle closed, runner
close returned and creation/image identity checks passed. Fresh config and
installed package copy per sample; no retained owned processes. Exact tester
SHAe737a93cb11331b09b83113984b9d279aef6cab129ff590063fd21fdee845398 checked.
Protected packages before/after match names/bytes/SHA256/last-write timestamps:
.20:25files/26854177B; .21:25files/26859809B. No accepted package writes.

Normal .21 median sampled memory was higher here; Metadata .21 median sampled
memory was lower, but its maximum private commit and median lifetime were
higher. All these rows remain included. This small fixed-order local sample
supports only these descriptive worker observations: no full-app/host/editor/
Tauri memory, general speed, allocation-to-RSS or broad improvement conclusion.

Closed evidence: native/artifacts/acceptance/drawio-worker-resources-fixed12-20261110,
266files/38890376B. Capture sibling:10files/146543B; command/streams/exits retained.
matrix-run.json162975B SHA36657fd632e8916ceeabdeeaf3dd6e9ce06e6d970cb87d24059f81c77b2e5ed3;
matrix-summary.json145115B SHAe93ceb9d54e65bc2d277c7821e1cfe553c17916a07a0cee3d963d3b2d62f1d47.
Sibling drawio-worker-resources-fixed12-20261110-inventory.json101448B SHA
6165d45fb07b6007bcde1cb456b509b722ade5978392b13abb48d2bf73f771c2 inventories
both closed roots outside itself. No further writes or requests. Resume only
read-only GPU bitmap lifecycle design; implementation awaits root activation.
