$readme = Get-Content -LiteralPath 'native/portable/README.md' -Raw
$base = (Resolve-Path -LiteralPath 'native/bin/levnote-native-worker-20261110.18').Path
$targets = [regex]::Matches($readme, '\[[^\]]+\]\(([^)]+)\)') | ForEach-Object { $_.Groups[1].Value } | Where-Object { $_ -notmatch '^(?:[a-z]+:|#)' } | Sort-Object -Unique
$missing = @()
foreach ($target in $targets) {
  $relative = $target.Split('#')[0]
  $resolved = Join-Path $base $relative
  $exists = Test-Path -LiteralPath $resolved
  '{0}`t{1}`t{2}' -f $target,$exists,$resolved
  if (-not $exists) { $missing += $target }
}
"targets=$($targets.Count) missing=$($missing.Count)"
if ($missing.Count -ne 0) { exit 1 }