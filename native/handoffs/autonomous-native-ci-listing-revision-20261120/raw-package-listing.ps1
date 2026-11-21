#requires -Version 7.0
# SOURCE ONLY / UNEXECUTED. Load after ci-lib.ps1 and before test-json-reducer.ps1.
Set-StrictMode -Version Latest
$script:ListingRevisionByteLimits = @{
    rawJsonl=256MB; rawPackageListing=32MB; packageListing=16MB; expectedManifest=4MB
    sourceInventory=16MB; toolchainIdentity=64KB; runIdentity=64KB
}
$script:ListingRevisionObjectBytes = 2MB
$script:ListingRevisionPackageCount = 4096
$script:ListingRevisionDepth = 64
$script:ListingRevisionFileCount = 2048
$script:ListingRevisionSourceCount = 65536

function ReadListingRevisionInput([string]$Path,[string]$SHA256,[string]$Name,[bool]$RetainBytes=$true) {
    SHA256Value $SHA256
    if (-not $script:ListingRevisionByteLimits.ContainsKey($Name)) { throw 'Unknown bounded input role.' }
    $maximum=$script:ListingRevisionByteLimits[$Name]
    $full=PlainFile $Path
    $before=Get-Item -LiteralPath $full -Force
    $length=$before.Length; $ticks=$before.LastWriteTimeUtc.Ticks
    # BEFORE allocation, hashing or decoding. The stream length is checked again
    # after opening with Windows write/delete sharing denied.
    if ($length -lt 1 -or $length -gt $maximum) { throw "Input byte limit/empty: $Name (maximum $maximum)." }
    $stream=[IO.File]::Open($full,[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::Read)
    $hasher=[Security.Cryptography.SHA256]::Create()
    $bytes=$null
    try {
        if ($stream.Length -ne $length -or $stream.Length -gt $maximum) { throw "Input changed before bounded read: $Name" }
        if ($RetainBytes) {
            $bytes=[byte[]]::new([int]$length)
            $offset=0
            while ($offset -lt $bytes.Length) {
                $read=$stream.Read($bytes,$offset,$bytes.Length-$offset)
                if (-not $read) { throw "Short bounded input read: $Name" }
                $offset+=$read
            }
            if ($stream.ReadByte() -ne -1) { throw "Input grew during bounded read: $Name" }
            $digest=$hasher.ComputeHash($bytes)
        } else {
            # Capture/final fingerprints use constant memory, with a length bound
            # on every read; ComputeHash(stream) alone would allow unbounded growth.
            $buffer=[byte[]]::new(32768); $total=0L
            while ($true) {
                $read=$stream.Read($buffer,0,$buffer.Length)
                if (-not $read) { break }
                $total+=$read
                if ($total -gt $length -or $total -gt $maximum) { throw "Input grew during bounded read: $Name" }
                $null=$hasher.TransformBlock($buffer,0,$read,$buffer,0)
            }
            if ($total -ne $length) { throw "Short bounded input read: $Name" }
            $null=$hasher.TransformFinalBlock([byte[]]::new(0),0,0)
            $digest=$hasher.Hash
        }
        $hash=[BitConverter]::ToString($digest).Replace('-','').ToLowerInvariant()
        $after=Get-Item -LiteralPath $full -Force; PlainChain $full
        if ($length -ne $after.Length -or $ticks -ne $after.LastWriteTimeUtc.Ticks -or $stream.Length -ne $length) {
            throw "Input changed during bounded read: $Name"
        }
        if ($hash -cne $SHA256) { throw "Pin mismatch: $Path" }
        $fact=[pscustomobject][ordered]@{path=''; absolutePath=$full; bytes=$length; sha256=$hash; lastWriteUtcTicks=$ticks}
        return [pscustomobject]@{fact=$fact; bytes=$bytes}
    } finally { $hasher.Dispose(); $stream.Dispose() }
}
function ListingRevisionUtf8([byte[]]$Bytes,[string]$Label) {
    if ($Bytes.Length -ge 3 -and $Bytes[0] -eq 239 -and $Bytes[1] -eq 187 -and $Bytes[2] -eq 191) { throw "UTF-8 BOM refused: $Label" }
    $encoding=[Text.UTF8Encoding]::new($false,$true)
    try { $null=$encoding.GetCharCount($Bytes) } catch { throw "Invalid UTF-8: $Label" }
    return $encoding
}
function ReadListingRevisionJson([byte[]]$Bytes,[string]$Label) {
    $encoding=ListingRevisionUtf8 $Bytes $Label
    return ,(StrictJson ($encoding.GetString($Bytes)))
}
function AssertListingRevisionCount([int]$Count,[int]$Maximum,[string]$Label) {
    if ($Count -gt $Maximum) { throw "Entry count limit: $Label (maximum $Maximum)." }
}
function ReadRawPackageObjects([byte[]]$Bytes) {
    $encoding=ListingRevisionUtf8 $Bytes 'rawPackageListing'
    $objects=[Collections.Generic.List[object]]::new()
    $stack=[byte[]]::new($script:ListingRevisionDepth)
    $depth=0; $start=-1; $quoted=$false; $escaped=$false
    # Linear scan of bounded bytes. UTF-8 continuation bytes cannot equal an ASCII
    # quote/backslash/delimiter. Strict JSON validates each complete slice next.
    for ($i=0; $i -lt $Bytes.Length; $i++) {
        $b=$Bytes[$i]
        if ($start -lt 0) {
            if ($b -eq 9 -or $b -eq 10 -or $b -eq 13 -or $b -eq 32) { continue }
            if ($b -ne 123) { throw 'Raw listing requires top-level objects and JSON whitespace only.' }
            if ($objects.Count -ge $script:ListingRevisionPackageCount) { throw 'Raw listing object count exceeds 4096.' }
            $start=$i; $depth=1; $stack[0]=123; $quoted=$false; $escaped=$false
            continue
        }
        if ($i-$start+1 -gt $script:ListingRevisionObjectBytes) { throw 'Raw listing object exceeds 2MiB.' }
        if ($quoted) {
            if ($escaped) { $escaped=$false }
            elseif ($b -eq 92) { $escaped=$true }
            elseif ($b -eq 34) { $quoted=$false }
            continue
        }
        if ($b -eq 34) { $quoted=$true; continue }
        if ($b -eq 123 -or $b -eq 91) {
            if ($depth -ge $script:ListingRevisionDepth) { throw 'Raw listing nesting exceeds depth 64.' }
            $stack[$depth]=$b; $depth++; continue
        }
        if ($b -eq 125 -or $b -eq 93) {
            $opener=if ($b -eq 125) { 123 } else { 91 }
            if ($depth -le 0 -or $stack[$depth-1] -ne $opener) { throw 'Mismatched delimiter in raw package listing.' }
            $depth--
            if ($depth -eq 0) {
                $text=$encoding.GetString($Bytes,$start,$i-$start+1)
                # JsonDocument's default max depth is also 64. Duplicate properties,
                # including case aliases in ignored nested fields, are refused.
                try { $object=StrictJson $text } catch {
                    if ($_.Exception.Message.StartsWith('Duplicate/aliased JSON property:',[StringComparison]::Ordinal)) { throw }
                    throw 'Malformed JSON object in raw package listing.'
                }
                $objects.Add($object); $start=-1
            }
        }
    }
    if ($start -ge 0) { throw 'Truncated raw package listing object.' }
    if (-not $objects.Count) { throw 'Raw package listing contains no objects.' }
    return ,([object[]]$objects.ToArray())
}
function AssertRawFieldCase([Collections.IDictionary]$Object,[string[]]$Known,[string]$Label) {
    $canonical=[Collections.Generic.Dictionary[string,string]]::new([StringComparer]::OrdinalIgnoreCase)
    foreach ($name in $Known) { $canonical.Add($name,$name) }
    foreach ($name in $Object.PSBase.Keys) {
        if ($canonical.ContainsKey($name) -and $name -cne $canonical[$name]) { throw "Mis-cased $Label property: $name" }
    }
}
function RequireRawField([Collections.IDictionary]$Object,[string]$Name,[string]$Label) {
    if (-not $Object.Contains($Name)) { throw "Missing raw $Label field: $Name" }
}
function RawWindowsSpelling([object]$Value,[string]$Label) {
    # Pure lexical validation. Never LocalPath/Join-Path/Get-Item/Resolve-Path on
    # a raw Dir/Module field; these values never become filesystem read targets.
    if ($Value -isnot [string] -or $Value -cnotmatch '^[A-Za-z]:[\\/]' -or $Value -match '[\x00-\x1F\x7F]') { throw "Invalid lexical Windows path: $Label" }
    $path=$Value.Replace('\','/')
    if ($path.Length -gt 3) {
        foreach ($part in $path.Substring(3).Split('/')) {
            if ($part -in @('','.', '..') -or $part -match '[:*?"<>|]' -or $part.EndsWith('.') -or $part.EndsWith(' ') -or
                $part -match '^(?i:CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³])(?:\.|$)') { throw "Invalid lexical Windows path: $Label" }
        }
    }
    return $path
}
function RawFalseFlag([Collections.IDictionary]$Object,[string]$Name,[string]$Label) {
    if ($Object.Contains($Name) -and ($Object[$Name] -isnot [bool] -or $Object[$Name])) { throw "Raw $Label requires absent/false $Name." }
}
function RawTestFiles([Collections.IDictionary]$Object,[string]$Field,[string]$Package,[string]$Relative,
    [Collections.Generic.HashSet[string]]$Seen,[Collections.Generic.HashSet[string]]$SourcePaths) {
    if (-not $Object.Contains($Field)) { return ,([object[]]@()) }
    $files=$Object[$Field]
    RequireArray $files "raw $Field"
    AssertListingRevisionCount $files.Count $script:ListingRevisionFileCount "raw $Field"
    foreach ($file in $files) {
        if ($file -isnot [string]) { throw "Raw test filename must be string: $Package / $Field" }
        AssertRelativeGrammar $file
        if ($file.Contains('/') -or -not $file.EndsWith('_test.go',[StringComparison]::Ordinal) -or -not $Seen.Add($file)) {
            throw "Raw test filename invalid/aliased: $Package / $Field"
        }
        if (-not $SourcePaths.Contains('native/'+$Relative+'/'+$file)) { throw "Raw test filename absent from source inventory: $Package / $file" }
    }
    return ,$files
}
function AssertOrdinalFileSet([object[]]$Raw,[object[]]$Normalized,[string]$Field,[string]$Package) {
    $left=[string[]]@($Raw); $right=[string[]]@($Normalized)
    [Array]::Sort($left,[StringComparer]::Ordinal); [Array]::Sort($right,[StringComparer]::Ordinal)
    if ($left.Count -ne $right.Count) { throw "Raw/normalized $Field differ: $Package" }
    for ($i=0; $i -lt $left.Count; $i++) {
        if ($left[$i] -cne $right[$i]) { throw "Raw/normalized $Field differ: $Package" }
    }
}
function AssertRawPackageListing([byte[]]$Bytes,[string]$RepositoryRoot,
    [Collections.Generic.HashSet[string]]$SourcePaths,[object[]]$Normalized,[object[]]$Expected) {
    $moduleRoot=(RawWindowsSpelling $RepositoryRoot 'source.repositoryRoot').TrimEnd([char]'/')+'/native'
    $rows=ReadRawPackageObjects $Bytes
    $raw=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    $aliases=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
    $packageFields=@('Dir','ImportPath','ImportComment','Name','Doc','Target','Shlib','Root','ConflictDir','ForTest','Export','BuildID','Module','Match','Goroot','Standard','DepOnly','BinaryOnly','Incomplete','DefaultGODEBUG','Stale','StaleReason','GoFiles','CgoFiles','CompiledGoFiles','IgnoredGoFiles','InvalidGoFiles','IgnoredOtherFiles','CFiles','CXXFiles','MFiles','HFiles','FFiles','SFiles','SwigFiles','SwigCXXFiles','SysoFiles','EmbedPatterns','EmbedFiles','CgoCFLAGS','CgoCPPFLAGS','CgoCXXFLAGS','CgoFFLAGS','CgoLDFLAGS','CgoPkgConfig','Imports','ImportMap','Deps','Error','DepsErrors','TestGoFiles','TestImports','TestEmbedPatterns','TestEmbedFiles','XTestGoFiles','XTestImports','XTestEmbedPatterns','XTestEmbedFiles')
    $moduleFields=@('Path','Version','Query','Versions','Replace','Time','Update','Main','Indirect','Dir','GoMod','GoVersion','Retracted','Deprecated','Error','Sum','GoModSum','Origin','Reuse')
    foreach ($row in $rows) {
        AssertRawFieldCase $row $packageFields 'package'
        foreach ($field in @('ImportPath','Dir','Module')) { RequireRawField $row $field 'package' }
        RequireString $row.ImportPath 'raw ImportPath'
        $name=$row.ImportPath
        if ($raw.ContainsKey($name)) { throw "Duplicate raw package identity: $name" }
        if (-not $aliases.Add($name)) { throw "Windows case alias raw package identity: $name" }
        $prefix='levnote.local/native/'
        if (-not $name.StartsWith($prefix,[StringComparison]::Ordinal)) { throw "Raw package module prefix differs: $name" }
        $relative=$name.Substring($prefix.Length)
        TestPackageIdentity $name $relative
        if ($row.Contains('ForTest') -and ($row.ForTest -isnot [string] -or $row.ForTest -cne '')) { throw 'Raw package ForTest rows unsupported.' }
        foreach ($flag in @('DepOnly','Incomplete','Goroot','Standard','BinaryOnly')) { RawFalseFlag $row $flag 'package' }
        if ($row.Contains('Error') -or $row.Contains('DepsErrors')) { throw 'Raw package Error/DepsErrors rows refused.' }
        if ((RawWindowsSpelling $row.Dir 'Dir') -cne ($moduleRoot+'/'+$relative)) { throw "Raw package Dir differs: $name" }
        if ($row.Module -isnot [Collections.IDictionary]) { throw 'Raw Module must be object.' }
        $module=$row.Module
        AssertRawFieldCase $module $moduleFields 'module'
        foreach ($field in @('Path','Main','Dir','GoMod')) { RequireRawField $module $field 'module' }
        if ($module.Path -isnot [string] -or $module.Path -cne 'levnote.local/native') { throw 'Raw Module.Path differs.' }
        if ($module.Main -isnot [bool] -or -not $module.Main) { throw 'Raw Module.Main must be true.' }
        if ($module.Contains('Replace') -or $module.Contains('Error')) { throw 'Raw module replacement/error unsupported.' }
        RawFalseFlag $module 'Indirect' 'module'
        if ($module.Contains('Version') -and ($module.Version -isnot [string] -or $module.Version -cne '')) { throw 'Raw main-module version unsupported.' }
        if ($module.Contains('GoVersion')) { RequireString $module.GoVersion 'raw Module.GoVersion' }
        if ((RawWindowsSpelling $module.Dir 'Module.Dir') -cne $moduleRoot) { throw 'Raw Module.Dir differs; replacement/workspace layout unsupported.' }
        if ((RawWindowsSpelling $module.GoMod 'Module.GoMod') -cne ($moduleRoot+'/go.mod')) { throw 'Raw Module.GoMod differs; replacement/workspace layout unsupported.' }
        $seen=[Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
        $internal=RawTestFiles $row 'TestGoFiles' $name $relative $seen $SourcePaths
        $external=RawTestFiles $row 'XTestGoFiles' $name $relative $seen $SourcePaths
        $raw.Add($name,[pscustomobject]@{Package=$name; RelativeDirectory=$relative; TestGoFiles=$internal; XTestGoFiles=$external; HasTests=($internal.Count+$external.Count -gt 0)})
    }
    $normalizedByName=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    $expectedByName=[Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($row in $Normalized) { $normalizedByName.Add($row.Package,$row) }
    foreach ($row in $Expected) { $expectedByName.Add($row.Package,$row) }
    $names=[string[]]@($raw.Keys); [Array]::Sort($names,[StringComparer]::Ordinal)
    foreach ($name in $names) {
        if (-not $normalizedByName.ContainsKey($name)) { throw "Raw package omitted from normalized listing: $name" }
    }
    # Companion rows were already checked as identical, unique and ordinal-sorted.
    foreach ($row in $Normalized) {
        $name=$row.Package
        if (-not $raw.ContainsKey($name)) { throw "Normalized package absent from raw listing: $name" }
        $projected=$raw[$name]
        if ($projected.RelativeDirectory -cne $row.RelativeDirectory) { throw "Raw/normalized RelativeDirectory differs: $name" }
        if ($projected.HasTests -ne $expectedByName[$name].HasTests) { throw "Raw/expected HasTests differs: $name" }
        AssertOrdinalFileSet $projected.TestGoFiles $row.TestGoFiles 'TestGoFiles' $name
        AssertOrdinalFileSet $projected.XTestGoFiles $row.XTestGoFiles 'XTestGoFiles' $name
    }
}
