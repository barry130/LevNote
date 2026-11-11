# Second fixed engine .6 delivery

Status: ROOT ACCEPTED EXACT .6 PACKAGE, final source documentation complete,
2026-11-09. Current `native/bin/levnote-native-worker-20261109.6` contains the
native host and two fixed one-shot export workers. Root read both actual
HostPath results, manifests/shipped README, original validator and sharing
failure, corrected source/aggregate/streams and accepted12/12. Source launch
docs now point to `.6`; product, packages and accepted evidence remain frozen.
No full UI or resource run belongs to this gate.

## Actual execution

Evidence directory:
[mindmap-engine-20261109-065002-package](../artifacts/acceptance/mindmap-engine-20261109-065002-package/).
Root explicitly activated delivery after reading the actual command gate:
9 RUN/PASS, command package0.537s/vet0,20 unchanged source/dependency inputs.
The activation context records exact accepted result/test/vet/source hashes;
the earlier command owner's hand-authored preliminary summaries are not used.

One fresh package invocation built Markdown worker, Mindmap worker and host
through the frozen shared script. All three build exits0 and streams empty.
One tester was compiled into a complete disposable21-file copy; its one
invocation ran only these two entries, once each:

- `TestMindmapWorkerExportBundledHostPathAcceptance`: actual PASS0.17s,
 704-byte SVG SHA256
 `1acaa480668668dcca201c57d0bc61e0876825aad78b79f46d5fa6dc49b99128`,
 3 exact whole-document warnings,2 dialogs and2 worker requests. Independent
 XML geometry/label/colors/escaping/image placeholder, pure parity and exact
 BOM/CRLF/source/metadata/history/selection/baseline oracles passed.
- `TestWorkerExportBundledHostPathAcceptance`: actual PASS0.13s,
 331-byte HTML SHA256
 `18fbe1f3be429c7af1f975e208933f08e154a6c2396ff75cdab61d516356b758`,
 1 warning and2 dialogs. Existing independent HTML/source/BOM/state oracles passed.

Each performs successful publication then exclusive-target refusal, so total
four real worker requests. Both RuntimeJoined records are true under the
source-reviewed joined-return contract. SaveDialog is injected, with no separate
PID observer or physical packaged-host UI claim. Actual package/tester compile/
HostPath wrapper times5.1502791s/1.9284956s/1.1330645s; all exit0/empty stderr.
These are local single-run observations, not performance comparison results.

## Immutable final evidence and fingerprints

[Aggregate attempt02](../artifacts/acceptance/mindmap-engine-20261109-065002-package/aggregate-attempt02.json)
**PASS12/12**,21 files20,187,007 bytes:
`59CFC6F967968A91D272D91B12218EEDB739C8434B400D77ACD8647983FD39C2`.

[Validator attempt02](../artifacts/acceptance/mindmap-engine-20261109-065002-package/validator-attempt02.json):
`C49D0E03590416DF8D2C66E068F099B96EDA1AF3B57A41E220CC54E80FC34802`.
[Validator source](../artifacts/acceptance/mindmap-engine-20261109-065002-package/aggregate-attempt02.ps1):
`335E3790F2DE32764FCA660AB50B3F2A7C10B89F948F9D9F1583359F1C6C0D60`.
[Original HostPath stdout](../artifacts/acceptance/mindmap-engine-20261109-065002-package/host-path.stdout.log),1851 bytes:
`2453FD14AB20D252B98950CA6AC1B5674BE1E6CD8A3A20BC23E853F276BF346D`.

| Payload | Bytes | SHA256 |
|---|---:|---|
| host | 13,706,752 | `8A4D96A0ACE3F6E50A98E38A9B38E2C8E6F2257C25BD5101606569BD12583310` |
| Markdown worker | 3,299,840 | `11BE430F7FBA5DE126DB8658893C9E9143189F4867C728437E13841C6D5E6F1D` |
| Markdown manifest | 337 | `F9CACEF76D5B82879347800D07FBC62511D84BE516103764E4C8E84D0B014C33` |
| Mindmap worker | 3,122,688 | `F36D89C941D4ACEB71EE97F083F5FF2237DD71CC2AAEC189AF4AAF9571E7A964` |
| Mindmap manifest | 328 | `F65A6E42F33CB0B2E0C9EDF19E4B91C5BB1615715CB3FCE350B7803ADFF1E2AC` |
| release manifest | 3,767 | `C83ECD84B9960C1ED33E870319D4662818EE386F10BDF3389733246775BBDDE2` |

Validator independently reads three Windows amd64 GUI PE headers, exact separate
fixed manifests,20 unique release entries,13 exact license/attribution assets,
substituted README/notice and19 local authored links. Candidate and disposable
copies remain identical after all requests. All1062 current build inputs,
15,657 toolchain files and733 protected historical files preserve membership,
bytes and hashes; protected files and both bundle copies also preserve UTC ticks.

## Preserved evidence-only failure

The original validator already wrote PASS12/12. Its final aggregate enumeration
then tried to hash its own still-open redirected `aggregate.stderr.log`; Windows
rejected the sharing mode. The original script, validator, exit1 command and
failure streams remain untouched. No production check failed and no product,
test, package or worker was rerun.

Root authorized a new `aggregate-attempt02.ps1`, new aggregate/validator names
and omission of only its current active stdout/stderr from recursive hashing.
The original harness still matches the baseline. After the correction process
exited, [standalone command/stream fingerprints](../artifacts/acceptance/mindmap-engine-20261109-065002-package/aggregate-attempt02-command.json)
recorded exit0,8.3450689s,266-byte stdout and empty stderr; record SHA256
`3E18E5B7A914CFEA53E9E9E5BF04912CBB4BE5812B06EE3D7181B38E2080BE5A`.
This is solely an evidence collection correction. Root has read and accepted
the original evidence plus the corrected aggregate; all remain immutable.

## Final source documentation and limits

Updated root `README.md`, `ROADMAP.md`, `CHANGELOG.md` and native `README.md`,
`VALIDATION.md`, `ARCHITECTURE.md`, `formats/API.md`, plus this handoff. The new
[root-acceptance record](../artifacts/acceptance/mindmap-engine-20261109-065002-package/root-acceptance.json)
uses its actual creation time and digests read from the accepted paths. Its
documentation inventory is a separate source snapshot for final root review,
not a claim that the shipped immutable README was replaced.

The shipped product is a native host plus two fixed export engines: Markdown
HTML accepts8MiB source, Mindmap SVG16MiB, each with64MiB output. The app and
runtime share one active slot across engines. Mindmap source mode needs no
View; preview/editing remain in the native host, as does draw.io SVG. Shared
Mindmap presentation helpers do no SVG work during Draw/sync. Whole-document
warnings do not claim node source coordinates; images remain placeholders,
excess warning metadata refuses, unknown source/metadata and encoding stay
unchanged. No arbitrary executable installation or sandbox is provided.

Final publication still uses synchronous exclusive WriteNewFile and cannot be
canceled after it starts. Program-folder portability does not move APPDATA
configuration. Old `.1/.2` allocation/process samples remain version-labeled;
there are no new resource, physical OSIME/GPU or complete UI startup results.
The next preview model stage requires separate root activation.
