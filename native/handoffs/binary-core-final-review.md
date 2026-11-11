# Binary core final review: focused corrections

ACTIVE2026-11-10 for existing binary GLM owner. Root read full frozen source and
tests. API direction accepted; stage acceptance pending these concrete fixes.
Do not start session/app yet. No further speculative API changes.

Three tests do not exercise their stated scenario:
- TestBinarySaveAsMidcopyCancellationLeavesNoDestination cancels before calling
  SaveAs, so it only tests entry rejection. Use a per-call actual writer wrapper
  seam which cancels after the first32KiB write, and assert nonzero partial write,
  no publication/temp leak, baseline unchanged. Also cancel immediately before
  publication and after commit (e.g. syncDir hook cancels then returns nil) to
  check Committed/Verified/Adopted and retained bytes.
- TestBinaryStableReadRejectsShortReadAndFinalCancellation returns an unrelated
  zero hash, so it tests identity mismatch only; its second half again cancels
  before entry. Add a per-call low-level bounded reader seam to the actual stable
  read helper, inject short EOF and read error, verify no baseline/snapshot is
  accepted. Cancel at the final check from that actual hook. Do not emulate the
  helper under test or add global mutable hooks. Existing high-level test seam
  can be removed if no longer necessary.
- TestBinarySameSizeRestoredMtimeReplacementIsRejected writes5bytes then6bytes.
  Use a different5byte value and assert same size/restored timestamp before the
  call so only digest can reveal the change. Also a replace-by-rename fixture
  tests identity separately where Windows permits ordinary replacement.

Add the assigned successful empty-file copy and copying the held snapshot after
source deletion. Check snapshot from another separately opened BinaryFile cannot
be adopted, even for identical bytes/path, before destination creation.

Small production correction: same-path SaveAs currently performs a full second
read into bytes that are discarded. Use bounded digest verification, retaining
no second source-size buffer, while preserving NoRewrite and cancellation. The
streaming digest helper already exists. Drop unused private verification/helper
fields if they become unnecessary. Preserve public signatures.

Verification/result currently lists commands without saved streams/exits or
file identities. Keep earlier actual tool-output failures as clearly historical
records if recoverable; never fabricate them or silently replace missing proof.
For these new corrections run focused regressions then one final affected
document suite/vet, capture exact wrapper arguments, output and real exit codes
in a fresh native/handoffs/binary-core-final-evidence-20261110 directory. Record
any earlier capture gap candidly. Freeze source identities and update result.
Do not repeat old passing checks merely to recreate logs without code correction.

## Final source review at 10:40: remaining test corrections

Root read current source and saved passing logs. Production direction is sound;
do not expand the API. Three assertions still do not prove the assigned boundary:

- PublicationAndPostCommitCancellation cancels a context discarded into `_`,
  then a custom verify hook manufactures context.Canceled for another context.
  Use ONE context passed to saveAs; syncDir cancels that context and returns nil.
  Keep the real default digest verifier (no verify hook). Assert errors.Is
  context.Canceled, committed=true, verified/adopted=false, destination retained,
  and the original Token still owned. This is the actual postcommit scenario.
- MidcopyCancellation counts calls but not actual successful bytes. Record the
  actual writer result, assert exactly one successful32KiB partial write and
  no more writes, and SameBaseline(oldToken), alongside existing no-temp checks.
- SameSizeRestoredMtime must assert size, timestamp AND file identity are equal
  after Chtimes before Snapshot; this isolates digest. Rename replacement should
  use identical bytes/size/restored timestamp and assert os.SameFile differs,
  so its detection is attributable to identity rather than another size/hash.

Correct these tests, preserve existing logs, then use new numbered filenames for
one focused affected run and one final document suite/vet. Do not overwrite prior
evidence. Freeze final identities/result. No other test expansion or session work
before acceptance.

## Rename fixture: Windows lazy identity trap found in final review

Current TestBinaryReplaceByRenameChangesIdentity incorrectly fails on
`!os.SameFile(before, after)`, thus REQUIRING the identities to match. Its green
result is explained by Go Windows `os.Stat` loading IDs lazily on first
SameFile: native/tools/go/src/os/types_windows.go loadFileId and sameFile.
Both path-based infos are first compared after rename, so both resolve the new
file. This is a fixture error, not evidence the actual BinaryFile baseline is
wrong: production captures handle-based File.Stat, which eagerly includes ID.

Fix only this remaining fixture and false result prose. Set the coarse timestamp
BEFORE OpenBinary. Capture before from the already eager `file.disk.info`.
Create identical bytes, rename replacement and restore the timestamp. Capture
after from an open replacement handle's File.Stat (close promptly). Assert
same size/time/bytes AND os.SameFile(before,after)==false. Then CheckDisk must
report Changed and Snapshot must reject ErrConflict while baseline is unchanged.
No skips or flips merely to make it pass. The supplied pinned Go source explains
why the earlier assertion passed. Postcommit correction is accepted by source
review; do not revise it. Run the one corrected test then one final affected
suite/vet with new evidence suffixes, retain previous records and document this
false-positive fixture clearly.
