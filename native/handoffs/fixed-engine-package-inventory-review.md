# Inventory decoder initial source review

2026-11-09. Root read landed package_inventory.go before its first gate. GLM
owns fixes/tests, Astra may consume names after correction. Concrete findings:

1. inventoryDeviceNumber first excludes every value except ASCII1..9, so later
   superscript logic is unreachable. COM¹, COM², COM³ and LPT¹/²/³ are Windows
   reserved device names and currently accepted. Use exact allowed suffix set
   and exercise each with and without extension.
2. inventoryValidPath currently permits < > " | ? * in components. These are
   illegal Windows file names, so reject for ALL entries, including ones not
   selected for copying. Include CONIN$/CONOUT$ device names and extension forms
   if relevant to the ordinary-disk policy. Prefix reserved base aliases such as
   'CON .txt' need a deliberate conservative refusal. Unknown names must not get
   cleaned into a usable path.
3. Case aliases currently key strings.ToLower. Consider Unicode simple-fold
   aliases (e.g. K vs Kelvin sign) as Windows alias risk: use bounded <=256entry
   EqualFold checks or document narrower ASCII admitted paths. Prefer retaining
   Unicode filenames with conservative EqualFold duplicate detection. Test an
   independent literal alias pair. Avoid claiming a complete adversarial NTFS
   normalization defense; fixed selected paths remain ASCII allowlisted.

Decoder structure/token exactness, hard total cap, integer ParseInt and no file
operations look appropriate. Apply these narrow fixes and meaningful tests,
gofmt owned files and run the planned focused gate. No need broaden to whole
runtime/process tests.

Test-body review before gate also found two incorrect or weak oracles:
- COM3D.txt is a normal filename, not COM3. Assert it is accepted (current test
  asserts rejection). COM10/LPT10 are also non-device filenames under this
  deliberately fixed1..9/superscript set; add positive controls.
- The surrogate test replaces text with `harm\\uDC00` in a Go raw literal,
  producing JSON escaped backslash and the letters uDC00. Refusal is then path
  backslash validation, not surrogate validation. Use a single JSON unicode
  escape inside descriptive scope/configuration where paths cannot mask the
  failure, and add valid surrogate-pair positive control. Likewise encode each
  invalid path via JSON marshaling when testing path validation; accidental
  JSON-invalid backslash is not coverage of decoded illegal filename handling.
