# Neutral native scene codec

Status: IMPLEMENTED; root read actual source/oracles/streams and accepted the
owned gate pending independent review,2026-11-09. Only new `core/enginescene`
source/tests changed. No catalog/IPC/runtime/worker/app/UI integration, build,
package, worker launch or resource measurement belongs to this stage.

## Contract and implementation

Public Scene/Node/Edge/Point/Rect and constants follow the corrected
[DTO contract](native-preview-dto-contract.md). `Encode(Scene)` returns one
deterministic JSON payload; `Decode([]byte)` accepts arbitrary field order.
Errors expose `ErrInvalidScene` (syntax/schema/semantics) and
`ErrLimitExceeded` (payload/string/count/numeric budgets), usable with errors.Is.
Errors return nil encoded bytes or zero decoded Scene. No caller input aliases
or mutable source-format/UI objects are retained; callers must not concurrently
mutate a Scene during Encode.

`scene.go` validates local string/paint/numeric budgets before graph allocation,
unique opaque IDs, exactly one root, required references/parents and n−1 edges.
A bounded linked adjacency plus iterative queue proves rooted connectivity,
including refusal of a disconnected cycle. There is no depth recursion.

`decode.go` has only fixed object/array/field readers. Required exact-key masks
reject duplicates, unknown/case-altered/missing fields, null and wrong shapes.
The payload cap is checked before scanning. Decoded strings use a fixed515-byte
scratch and check each UTF-8 rune/escape before copying; lone surrogates and
invalid raw UTF-8 refuse. Arrays stop before parsing the first excess element;
declared counts never allocate arrays, including when they precede the data.
Geometric slice capacity is clamped to the hard element count. Global tree
invariants run after bounded parsing, before publication.

Number scanning accepts ordinary JSON exponent/fraction spellings without
whole-token allocation or big.Int/Rat. Saturated exponent and first/last nonzero
positions decide exact integer counts and power-of-ten bounds; a rounded
fraction cannot become an accepted count. At most800 significant digits plus a
sticky nonzero tail reach strconv, matching the pinned conversion's bounded
decimal precision. Signed zero is retained; positive dimensions that underflow
to zero refuse. Output fields are float64, not arbitrary-precision decimals.

`encode.go` shares one deterministic emitter between bounded count and exact
fill. UTF-8 text is directly JSON-escaped; no complete escaped temporary string
is created. Success returns len==cap output at most32MiB, not a process-memory
limit. No public custom budget seam was added. The neutral DTO carries no
worker/source identity, raw source, image data, callback, or publication path.

## Substantive checks and actual evidence

Twelve top-level tests in `scene_test.go`, `numeric_test.go`, `admission_test.go`:

- Authored literal wire/expected DTO and deterministic output, Unicode/JSON
  escapes including backslash+slash/control/NUL, arbitrary object order and late
  counts, all four layouts, exact normalized/theme-empty paints; caller input
  mutation cannot alter decoded values and encoding cannot alter the scene.
- Exact512-byte IDs/515-byte labels, missing/duplicate/escaped-duplicate/null/
  unknown/case-altered keys, malformed shapes/UTF-8/surrogates/numbers, every
  truncated prefix and trailing data. Nested edge/point requirements are tested.
- Tree identity/degree/reference errors, root-parent/self-edge/disconnected
  cycles; ten-million dimensions/billion coordinates, finite/positive checks.
- Independent IEEE expected bits for ordinary exponent, minimum subnormal,
  signed zero, underflow, long coefficient canceled by exponent, exact midpoint
  and above/below midpoint with tails beyond800 digits. Exact fractional-count
  refusal and long saturating/leading-zero exponents have separate oracles.
- Reader-position/retained-count oracles prove early string and array admission,
  not merely final refusal. A100,000-node/99,999-edge chain proves nonrecursive
  processing. Real32MiB payload/output succeeds; one extra byte refuses with
  no partial publication. No claim of allocator or process-resource measurement.

One actual run, no failed execution:
[owned-attempt01](../artifacts/acceptance/native-scene-20261109-073344-owned-attempt01/).
Formatting ran before the before/after inventory. Exact helper clears inherited
LEVNOTE opt-ins and GOFLAGS, then uses explicit wrapper arrays:

```powershell
./scripts/go.ps1 -GoArguments @('test','./core/enginescene','-count=1','-timeout','120s','-v')
./scripts/go.ps1 -GoArguments @('vet','./core/enginescene')
```

Actual12 top-level RUN/PASS; package **1.240s**, outer test2.5028303s and
vet0.881761s. Both exit0; stderr empty, vet stdout empty. All12 inventoried source,
module/wrapper/toolchain/conversion-reference files unchanged during test/vet.
Root reviewed actual run/go-check scripts and raw stdout; no test rerun followed.

| Record | SHA256 |
| --- | --- |
| result.json | `944C4D02BD7C629D6ADC2194EFC0D8AE23131177861292D32E79B90156E90E53` |
| source-before/source-after.json | `F36D4F4AF0413FBDD09E026C6021457F8C42A30B6022CEC09AAAAECBD42BDCBC` |
| test.stdout.log | `60573526CBF7A3D9EB9E83EE1BB716E7E87EEBC74EAB0A776625BBFFEDFA276C` |
| scene.go | `DC252C6774ECCF99B3A94969BD6884B46AB6CBB2284D1A82A0A0FE23F8D89E73` |
| decode.go | `73AB1613CB66E64110334CF441F95AA721B8B4B91ABDF4A79C238795C6416148` |
| encode.go | `1721531325F97A29A6B6B8981F63FF04043B6EE4C14650AFC6ED4A375F7DB36F` |

All six source/test hashes are in the retained inventory. Current accepted `.6`
bundle and historical evidence remain unchanged. The following producer/IPC/
host consumer stages require separate root activation.
