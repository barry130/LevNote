# Draw.io custom-suffix route implementation result

FROZEN FOR INDEPENDENT REVIEW 2026-11-10. This is workspace source only. It
does not describe or modify the accepted `.19` binary/package, and it is not a
`.20` delivery or acceptance.

## Outcome

An installed schema3 JSON configuration can now use exact
`"nativeRoute":"builtin.drawio"` to route a custom suffix through the shipped
native draw.io editor. The route inherits the existing selected-page SVG worker
contract (`levnote.drawio-svg`, `svg`, `RequiresView:true`) without inventing an
outline or scene capability. Unknown, wrong-case, leading-space and
trailing-space route spellings remain rejected.

The app now describes schema3 routes as the current bundled native format's
capability instead of promising edit/outline/export for every route. Its
install dialog lists Markdown, mind maps and draw.io. Disabling builtin
`drawio` does not disable an independently enabled installed draw.io route;
disabling or losing the compiled route falls back visibly to safe source /
readonly inspection.

## Changed files

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/core/plugins/native_route.go` | 3,915 | `24F7108DB4423AA3C8EB9C50BD0123844AF6088216837DD21CF6323C41B41DB6` |
| `native/core/plugins/declarative.go` | 18,002 | `405AB8406B655C9E3DF8C514422B745D2322EB432F0C28175304CDC8CEF614F8` |
| `native/core/plugins/schema3_test.go` | 19,413 | `C1D451F459E8710DE94A68997A04C7172BC45ACFDCAA724D37ABD9D73694D028` |
| `native/modules/modules.go` | 1,997 | `E8438F294647E0C73522C48141B05DBA0D5DA8BEECCD8B2663C2520FE3CFFCFC` |
| `native/modules/native_route_test.go` | 6,578 | `2A6C3078393BDED0DE90BAB79F7E41278FCD53BF27CAC244C90681175D316BC8` |
| `native/app/plugins.go` | 10,836 | `91D717983A1CAA462D5977572C83BEF0F99A24FF59E275E5C81EA3BDC777D476` |
| `native/app/declarative_outline_test.go` | 4,431 | `5E6D3662D26A7F149FA23EAF633D21C7F9120601CDE84DE175557AF0039AD2E7` |
| `native/app/drawio_route_test.go` | 13,672 | `D81CA768DD4A8A3FCFC019978ED89619FD06DB9D5998894CA3FD327DA6274F5F` |

## Covered behavior

- Exact plugin decode, normalized extension, manager install/enable, and
  independent builtin disable. The “unknown route” refusal now uses
  `builtin.unknown`; draw.io wrong-case and whitespace variants are separately
  red.
- The module factory returns the existing `graphview.DrawioAdapter()` template,
  `svg` export and view-required worker capability with no `Outline` or
  `SceneEngine`. Capability slices/pointers remain fresh per call.
- New-from-plugin creates the existing valid draw.io page template. The parser
  reports only its existing structural-container warnings for cells `0` and
  `1`; the page itself parses as `page-1` / `页面 1`.
- A custom suffix opens in preview, source and split. BOM, CRLF, unknown XML
  attributes and source bytes survive unchanged save, mapped-suffix SaveAs,
  original preservation and reopen.
- A readonly custom-suffix original remains readonly; SaveAs creates a writable
  exact-byte copy.
- A real second-page field composition is flushed as one intended transaction
  before preparation. The request carries `{Index:1, ID:"p2"}`, source equals
  the post-Flush buffer, version advances once, UndoDepth advances once,
  RedoDepth is zero, one Undo returns to baseline, and the disk original is
  unchanged.
- Source mode and a nil view visibly refuse draw.io export without invoking the
  runner.
- A real conflicting draw.io field draft blocks installed-route disable while
  retaining the owning view; after the authority change is corrected, the same
  draft flushes and is not lost.
- A same-ID manifest refresh cancels and replaces the old draw.io route; a late
  old export result is cleared and cannot open a dialog or publish a file.
- Builtin draw.io disable leaves the installed route active. Installed-route
  disable yields editable source, readonly retains inspection semantics, and a
  missing compiled adapter reports `配置指定的原生格式不可用`.
- The install-dialog message includes `Markdown / 思维导图 / draw.io`.

Inherited lower-layer conflict coverage remains
`native/modules/graphview/safety_test.go:
TestDrawioDraftTracksPageIDAcrossReorderAndBlocksChangedField`, including
page-ID tracking, field conflict retention, refused page change and
`ErrDraftConflict`. No duplicate worker execution was performed.

## Commands and evidence

All Go commands used `native/scripts/go.ps1` from the repository root. Raw
streams and exit files are under
`native/handoffs/drawio-route-implementation-evidence/`.

1. The first evidence harness used a relative Tee path. Since the wrapper enters
   `native/`, it resolved to a nonexistent doubled path before a test result was
   obtained. This is recorded in
   `focused-development-0-harness-failure.log`.
2. Focused development check, exit `1`:

   ```powershell
   .\native\scripts\go.ps1 -GoArguments @('test','./core/plugins','./modules','./app','-count=1','-run','^(TestSchema3(RoutesAndOmittedDefaults|StrictAdmission|DrawioRouteInstalledLifecycle)|TestNativeDrawioRouteCapabilitiesAndTemplate|TestNativeRouteAdaptersOwnMutableCapabilities|TestDrawioRoute|TestDeclarativeCapability.*)$')
   ```

   This red run found two test-oracle mistakes only: the existing template
   intentionally emits structural-container warnings, and the enablement
   fixture had not disabled builtin `drawio`. It is preserved as
   `focused-development-1.log`, SHA-256
   `AAB42CA24F041025E6E319677E1E0B358DAD34845A88402EDBCE9E8F196CAE4E`.
3. Corrected focused check, exit `0`, with a corrected `TestDrawioRoute.*`
   pattern:

   ```powershell
   .\native\scripts\go.ps1 -GoArguments @('test','./core/plugins','./modules','./app','-count=1','-run','^(TestSchema3(RoutesAndOmittedDefaults|StrictAdmission|DrawioRouteInstalledLifecycle)|TestNativeDrawioRouteCapabilitiesAndTemplate|TestNativeRouteAdaptersOwnMutableCapabilities|TestDrawioRoute.*|TestDeclarativeCapability.*)$')
   ```

   Result: `core/plugins` 0.616s, `modules` 0.781s, `app` 2.006s. Raw log
   SHA-256 `61FABCF23066ADDCF1E480B1842225DECD1DAB500C96BB080751B9297B41DFE9`;
   exit file records `0`.
4. Final affected suite, exit `0`:

   ```powershell
   .\native\scripts\go.ps1 -GoArguments @('test','./core/plugins','./modules','./app','-count=1')
   ```

   Result: `core/plugins` 2.171s, `modules` 0.877s, `app` 16.709s. Raw log
   SHA-256 `4EFE05254D9347EF4B9AD81348F12C8EAB600B67CCD6BF2C0CF0A432CE7C2827`;
   exit file records `0`.
5. Final affected vet, exit `0`, no stdout/stderr:

   ```powershell
   .\native\scripts\go.ps1 -GoArguments @('vet','./core/plugins','./modules','./app')
   ```

   Raw log is zero bytes with SHA-256
   `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`;
   exit file records `0`.

## Limits

No package or binary was built; no worker, preview worker or host was launched;
no resource/UI timing benchmark was run. `.19` package/evidence and graphview
production were not changed. No 760x480 capture was produced because the change
is exercised through the real app/Tester state and this stage did not require a
visual claim. The tests do not prove physical Windows input, IME behavior,
worker process isolation, package acceptance or custom-suffix behavior inside
the immutable `.19` binary.
