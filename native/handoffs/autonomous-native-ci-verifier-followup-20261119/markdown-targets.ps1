#requires -Version 7.0
# Dedicated authored-package Markdown target policy; ci-lib.ps1 supplies filesystem guards.
Set-StrictMode -Version Latest
function ParseMarkdownTarget(
    [string]$RawTarget,
    [ValidateSet('Auto','File','Directory')][string]$Intent='Auto'
) {
    if ([string]::IsNullOrEmpty($RawTarget)) { throw 'Empty Markdown target.' }
    # Preserve the old classification boundary: only http(s): and a LEADING # are
    # non-filesystem targets. Never split/decode a local query, fragment or URL escape.
    if ($RawTarget -match '^(https?:|#)') {
        if ($Intent -ne 'Auto') { throw 'Filesystem intent attached to non-filesystem target.' }
        $kind=if ($RawTarget.StartsWith('#')) { 'Fragment' } else { 'External' }
        return [pscustomobject]@{kind=$kind; raw=$RawTarget; relative=$null; explicitDirectoryIntent=$false}
    }
    AssertRelativeGrammar $RawTarget $true # RAW validation must precede Substring.
    $directory=$RawTarget.EndsWith('/',[StringComparison]::Ordinal)
    if (($Intent -eq 'File' -and $directory) -or ($Intent -eq 'Directory' -and -not $directory)) {
        throw 'Markdown spelling disagrees with explicit file/directory intent.'
    }
    if ($directory) {
        # Exactly ONE slash was admitted by the grammar. Do not use TrimEnd.
        $relative=$RawTarget.Substring(0,$RawTarget.Length-1)
        return [pscustomobject]@{kind='Directory'; raw=$RawTarget; relative=$relative; explicitDirectoryIntent=$true}
    }
    return [pscustomobject]@{kind='File'; raw=$RawTarget; relative=$RawTarget; explicitDirectoryIntent=$false}
}
function ResolveMarkdownTarget(
    [string]$PackageRoot,[string]$RawTarget,
    [ValidateSet('Auto','File','Directory')][string]$Intent='Auto'
) {
    $target=ParseMarkdownTarget $RawTarget $Intent
    if ($target.kind -in @('Fragment','External')) { return $target }
    $root=PlainDirectory $PackageRoot
    if ($target.kind -ceq 'File') {
        # Keep the complete file-only chain; RelativeFile never accepts licenses/.
        $full=PlainFile (RelativeFile $root $target.relative)
        PlainChain $full
        if (-not (Within $full $root)) { throw 'Markdown file escapes package root.' }
    } else {
        $full=LocalPath (Join-Path $root $target.relative)
        if (-not (Within $full $root)) { throw 'Markdown directory escapes package root.' }
        $full=PlainDirectory $full
        PlainChain $full
        if (-not (Within $full $root)) { throw 'Markdown directory escapes package root.' }
    }
    return [pscustomobject]@{kind=$target.kind; raw=$target.raw; relative=$target.relative; absolutePath=$full; explicitDirectoryIntent=$target.explicitDirectoryIntent}
}
function CheckMarkdownLinks([string]$PackageRoot,[string]$Text,[string]$DocumentName) {
    # Deliberately bounded syntax, identical inputs for authored portable README and
    # its generated package README. Not a CommonMark/reference-link/HTML parser.
    $matches=[regex]::Matches($Text,'\[(?<label>[^\[\]\r\n]*)\]\((?<target>[^()\r\n]*)\)')
    if ([regex]::Matches($Text,'\]\(').Count -ne $matches.Count) { throw "Unsupported/malformed inline link syntax: $DocumentName" }
    $facts=[Collections.Generic.List[object]]::new()
    foreach ($match in $matches) {
        $resolved=ResolveMarkdownTarget $PackageRoot $match.Groups['target'].Value
        $facts.Add([ordered]@{document=$DocumentName; offset=$match.Index; label=$match.Groups['label'].Value; target=$resolved})
    }
    return $facts.ToArray()
}
