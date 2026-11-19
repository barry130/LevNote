package document

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBinarySnapshotExactBytesAndSafeStreams(t *testing.T) {
	original := append([]byte("binary-prefix"), 0, 1, 2, 255, 'z')
	path := filepath.Join(t.TempDir(), "picture.png")
	put(t, path, original)

	file, snapshot, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	if file.Path() != path || file.Kind() != BinaryKindPNG || file.Size() != int64(len(original)) {
		t.Fatal(file.Path(), file.Kind(), file.Size())
	}
	if file.SourceReadOnly() || file.Revision() != 1 {
		t.Fatal(file.SourceReadOnly(), file.Revision())
	}
	if snapshot.Size() != int64(len(original)) || !file.SameBaseline(snapshot.Token()) {
		t.Fatal(snapshot.Size(), snapshot.Token().Valid())
	}

	var streamed bytes.Buffer
	if written, err := snapshot.WriteTo(&streamed); err != nil || written != int64(len(original)) || !bytes.Equal(streamed.Bytes(), original) {
		t.Fatal(written, err, streamed.Len())
	}
	readBytes, err := io.ReadAll(snapshot.Reader())
	if err != nil || !bytes.Equal(readBytes, original) {
		t.Fatal(len(readBytes), err)
	}
	if _, ok := snapshot.Reader().(*bytes.Reader); ok {
		t.Fatal("public reader exposed bytes.Reader")
	}
	piece := make([]byte, 4)
	if n, err := snapshot.ReaderAt().ReadAt(piece, 5); err != nil || n != len(piece) || !bytes.Equal(piece, original[5:9]) {
		t.Fatal(n, err, piece)
	}
	if n, err := snapshot.ReaderAt().ReadAt(piece, int64(len(original))); err != io.EOF || n != 0 {
		t.Fatal(n, err)
	}
}

type mutatingWriter struct {
	calls int
}

func (w *mutatingWriter) Write(p []byte) (int, error) {
	w.calls++
	if len(p) == 0 {
		return 0, nil
	}
	p[0] ^= 0xff
	return len(p), nil
}

func TestBinaryWriteToDoesNotExposePrivateStorage(t *testing.T) {
	original := []byte("private bytes that must not alias a writer")
	path := filepath.Join(t.TempDir(), "data.bin")
	put(t, path, original)
	_, snapshot, err := OpenBinary(context.Background(), path, BinaryKindUnknown)
	if err != nil {
		t.Fatal(err)
	}
	writer := &mutatingWriter{}
	if written, err := snapshot.WriteTo(writer); err != nil || written != int64(len(original)) || writer.calls == 0 {
		t.Fatal(written, err, writer.calls)
	}
	var copied bytes.Buffer
	if _, err = snapshot.WriteTo(&copied); err != nil || !bytes.Equal(copied.Bytes(), original) {
		t.Fatal(err, copied.String())
	}
}

func TestBinarySnapshotRefusesChangedMissingAndSupportsTokenRead(t *testing.T) {
	original := []byte("original-binary")
	path := filepath.Join(t.TempDir(), "image.png")
	put(t, path, original)
	file, snapshot, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	token := file.Token()
	if !token.Valid() || token.Path() != path || token.Kind() != BinaryKindPNG || token.Size() != int64(len(original)) {
		t.Fatal(token.Valid(), token.Path(), token.Kind(), token.Size())
	}
	changed := append([]byte(nil), original...)
	changed[len(changed)-1] = 'X'
	put(t, path, changed)
	if _, err = file.Snapshot(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	if _, err = ReadBinarySnapshot(context.Background(), token); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	var old bytes.Buffer
	if _, err = snapshot.WriteTo(&old); err != nil || !bytes.Equal(old.Bytes(), original) {
		t.Fatal(err, old.String())
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err = file.Snapshot(context.Background()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	put(t, path, []byte("new-binary"))
	reloaded, err := file.Reload(context.Background())
	if err != nil || file.Revision() != 2 || !file.SameBaseline(reloaded.Token()) {
		t.Fatal(err, file.Revision(), file.SameBaseline(reloaded.Token()))
	}
	if file.SameBaseline(snapshot.Token()) {
		t.Fatal("reload did not invalidate the old snapshot authority")
	}
	if !bytes.Equal(read(t, path), []byte("new-binary")) {
		t.Fatal("reloaded bytes changed")
	}
	var retained bytes.Buffer
	if _, err = snapshot.WriteTo(&retained); err != nil || !bytes.Equal(retained.Bytes(), original) {
		t.Fatal(err, retained.String())
	}
	tokenCopy, err := ReadBinarySnapshot(context.Background(), reloaded.Token())
	if err != nil || !file.SameBaseline(tokenCopy.Token()) {
		t.Fatal(err, file.SameBaseline(tokenCopy.Token()))
	}
}

func TestBinaryTokenReadIsConcurrentAndReadOnly(t *testing.T) {
	original := []byte("concurrent immutable token")
	path := filepath.Join(t.TempDir(), "photo.jpg")
	put(t, path, original)
	file, _, err := OpenBinary(context.Background(), path, BinaryKindJPEG)
	if err != nil {
		t.Fatal(err)
	}
	token := file.Token()
	results := make(chan []byte, 4)
	errs := make(chan error, 4)
	for range 4 {
		go func() {
			snapshot, err := ReadBinarySnapshot(context.Background(), token)
			if err != nil {
				errs <- err
				return
			}
			var output bytes.Buffer
			if _, err = snapshot.WriteTo(&output); err != nil {
				errs <- err
				return
			}
			results <- output.Bytes()
		}()
	}
	for range 4 {
		select {
		case data := <-results:
			if !bytes.Equal(data, original) {
				t.Fatal("concurrent token read changed bytes")
			}
		case err := <-errs:
			t.Fatal(err)
		}
	}
}

func TestBinaryOpenRejectsUnknownKindOversizeNonregularAndCancellation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.png")
	put(t, path, []byte("x"))
	if _, _, err := OpenBinary(context.Background(), path, BinaryKind("gif")); !errors.Is(err, ErrUnsupportedBinaryKind) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := OpenBinary(cancelled, path, BinaryKindPNG); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, _, err := OpenBinary(context.Background(), dir, BinaryKindUnknown); !errors.Is(err, ErrNotRegular) {
		t.Fatal(err)
	}
	largePath := filepath.Join(dir, "large.png")
	put(t, largePath, nil)
	if err := os.Truncate(largePath, MaxBinaryBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := OpenBinary(context.Background(), largePath, BinaryKindUnknown); !errors.Is(err, ErrTooLarge) {
		t.Fatal(err)
	}
}

func TestBinarySaveAsExactNonOverwriteAndSamePathNoRewrite(t *testing.T) {
	dir := t.TempDir()
	original := []byte("exact compressed bytes")
	source := filepath.Join(dir, "source.png")
	destination := filepath.Join(dir, "destination.png")
	put(t, source, original)
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := file.SaveAs(context.Background(), destination, snapshot); err != nil ||
		!result.Committed || !result.Verified || !result.Adopted || result.NoRewrite {
		t.Fatal(result, err)
	}
	if !bytes.Equal(read(t, destination), original) || !bytes.Equal(read(t, source), original) {
		t.Fatal("SaveAs changed source or destination bytes")
	}
	if file.Path() != destination || file.Revision() != 2 || file.SameBaseline(snapshot.Token()) {
		t.Fatal(file.Path(), file.Revision(), file.SameBaseline(snapshot.Token()))
	}
	var old bytes.Buffer
	if _, err = snapshot.WriteTo(&old); err != nil || !bytes.Equal(old.Bytes(), original) {
		t.Fatal(err, old.String())
	}
	current, err := file.Snapshot(context.Background())
	if err != nil || current.Size() != int64(len(original)) || !file.SameBaseline(current.Token()) {
		t.Fatal(err, current.Size(), file.SameBaseline(current.Token()))
	}

	existing := filepath.Join(dir, "existing.png")
	put(t, existing, []byte("someone else"))
	if result, err := file.SaveAs(context.Background(), existing, current); !errors.Is(err, ErrExists) ||
		result.Committed || result.Verified || result.Adopted || string(read(t, existing)) != "someone else" {
		t.Fatal(result, err, string(read(t, existing)))
	}

	stampBefore, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := file.SaveAs(context.Background(), destination, current); err != nil ||
		result.NoRewrite != true || result.Verified != true || result.Committed {
		t.Fatal(result, err)
	}
	stampAfter, err := os.Stat(destination)
	if err != nil || !stampAfter.ModTime().Equal(stampBefore.ModTime()) {
		t.Fatal(err, stampBefore.ModTime(), stampAfter.ModTime())
	}
}

func TestBinarySaveAsAllowsChangedSourceButNotStaleAuthority(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "old.png")
	destination := filepath.Join(dir, "copy.png")
	put(t, source, []byte("old accepted bytes"))
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	put(t, source, []byte("external replacement"))
	if result, err := file.SaveAs(context.Background(), destination, snapshot); err != nil ||
		!result.Committed || !result.Verified || !result.Adopted {
		t.Fatal(result, err)
	}
	if !bytes.Equal(read(t, destination), []byte("old accepted bytes")) {
		t.Fatal("SaveAs substituted changed source bytes")
	}
	if !bytes.Equal(read(t, source), []byte("external replacement")) {
		t.Fatal("SaveAs changed the original source")
	}
	current, err := file.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	currentToken := current.Token()

	staleDestination := filepath.Join(dir, "stale.png")
	if result, err := file.SaveAs(context.Background(), staleDestination, snapshot); !errors.Is(err, ErrStaleBinarySnapshot) ||
		result.Committed || result.Verified || result.Adopted {
		t.Fatal(result, err)
	}
	if _, err = os.Stat(staleDestination); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if !file.SameBaseline(currentToken) {
		t.Fatal("stale write invalidated the current baseline")
	}
}

func TestBinarySaveAsLosesExclusiveRaceWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	destination := filepath.Join(dir, "race.png")
	put(t, source, []byte("candidate"))
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	hookReached := false
	hooks := binarySaveHooks{
		beforeWrite: func() error {
			hookReached = true
			put(t, destination, []byte("winner changed"))
			return nil
		},
	}
	result, err := file.saveAs(context.Background(), destination, snapshot, hooks)
	if !errors.Is(err, ErrExists) || result.Committed || result.Verified || result.Adopted {
		t.Fatal(result, err)
	}
	if !hookReached || string(read(t, destination)) != "winner changed" {
		t.Fatal("exclusive publication overwrote a racing destination")
	}
}

func TestBinarySaveAsReportsPostCommitFailureWithoutAdoption(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	destination := filepath.Join(dir, "copy.png")
	original := []byte("committed bytes")
	put(t, source, original)
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	hooks := binarySaveHooks{syncDir: func(string) error { return os.ErrDeadlineExceeded }}
	result, err := file.saveAs(context.Background(), destination, snapshot, hooks)
	if err == nil || !result.Committed || result.Verified || result.Adopted || file.Path() != source {
		t.Fatal(result, err, file.Path())
	}
	if !bytes.Equal(read(t, destination), original) {
		t.Fatal("committed destination was removed after post-commit failure")
	}

	nextDestination := filepath.Join(dir, "next.png")
	hooks = binarySaveHooks{verify: func(context.Context, string) (identity, error) {
		return identity{}, os.ErrInvalid
	}}
	result, err = file.saveAs(context.Background(), nextDestination, snapshot, hooks)
	if err == nil || !result.Committed || result.Verified || result.Adopted || file.Path() != source {
		t.Fatal(result, err, file.Path())
	}
	if !bytes.Equal(read(t, nextDestination), original) {
		t.Fatal("verified-failure destination was removed")
	}
}

func TestBinarySaveAsPublicationAndPostCommitCancellation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	destination := filepath.Join(dir, "copy.png")
	original := []byte("cancel boundary")
	put(t, source, original)
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	originalToken := file.Token()

	beforePublishCtx, cancelBeforePublish := context.WithCancel(context.Background())
	hooks := binarySaveHooks{beforePublish: func() error {
		cancelBeforePublish()
		return nil
	}}
	if result, err := file.saveAs(beforePublishCtx, destination, snapshot, hooks); !errors.Is(err, context.Canceled) ||
		result.Committed || result.Verified || result.Adopted {
		t.Fatal(result, err)
	}
	if _, err = os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".levnote-save-") {
			t.Fatal("temporary file leaked before publication")
		}
	}

	afterCommitCtx, cancelAfterCommit := context.WithCancel(context.Background())
	hooks = binarySaveHooks{syncDir: func(string) error {
		cancelAfterCommit()
		return nil
	}}
	if result, err := file.saveAs(afterCommitCtx, destination, snapshot, hooks); !errors.Is(err, context.Canceled) ||
		!result.Committed || result.Verified || result.Adopted {
		t.Fatal(result, err)
	}
	if !bytes.Equal(read(t, destination), original) {
		t.Fatal("committed destination was not retained")
	}
	if file.Path() != source || !file.SameBaseline(originalToken) {
		t.Fatal(file.Path(), file.SameBaseline(originalToken))
	}
}

func TestAtomicOutcomePreservesExistingTextWrapperBehavior(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	put(t, path, []byte("before"))
	if err := writeAtomic(path, []byte("after"), 0600, false, nil); err != nil {
		t.Fatal(err)
	}
	if string(read(t, path)) != "after" {
		t.Fatal("atomic wrapper lost data")
	}
	outcome, err := writeAtomicWriterOutcome(path, 0600, false,
		func() error { return ErrConflict }, func(io.Writer) error { return nil }, nil)
	if !errors.Is(err, ErrConflict) || outcome.Committed || string(read(t, path)) != "after" {
		t.Fatal(outcome, err, string(read(t, path)))
	}
}

func TestBinarySnapshotRejectsNilAndMissingFileAuthority(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	put(t, path, []byte("data"))
	file, _, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.SaveAs(context.Background(), path, nil); !errors.Is(err, ErrNoBinaryBaseline) {
		t.Fatal(err)
	}
	missing := &BinaryFile{}
	if _, err = missing.Snapshot(context.Background()); !errors.Is(err, ErrNoBinaryBaseline) {
		t.Fatal(err)
	}
}

func TestBinarySaveAsMidcopyCancellationLeavesNoDestination(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "large.png")
	original := bytes.Repeat([]byte{0x42}, 96<<10)
	put(t, source, original)
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	originalToken := file.Token()
	ctx, cancel := context.WithCancel(context.Background())
	writes := 0
	writtenBytes := 0
	destination := filepath.Join(dir, "cancelled.png")
	hooks := binarySaveHooks{
		wrapWriter: func(writer io.Writer) io.Writer {
			return &cancelAfterFirstWrite{
				writer: writer, cancel: cancel,
				writes: &writes, writtenBytes: &writtenBytes,
			}
		},
	}
	if result, err := file.saveAs(ctx, destination, snapshot, hooks); !errors.Is(err, context.Canceled) ||
		result.Committed || result.Verified || result.Adopted {
		t.Fatal(result, err)
	}
	if writes != 1 || writtenBytes != 32<<10 {
		t.Fatal(writes, writtenBytes)
	}
	if _, err = os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".levnote-save-") {
			t.Fatal("temporary file leaked")
		}
	}
	if file.Path() != source || file.Size() != int64(len(original)) || !file.SameBaseline(originalToken) {
		t.Fatal(file.Path(), file.Size(), file.SameBaseline(originalToken))
	}
}

type cancelAfterFirstWrite struct {
	writer       io.Writer
	cancel       context.CancelFunc
	writes       *int
	writtenBytes *int
}

func (w *cancelAfterFirstWrite) Write(p []byte) (int, error) {
	if *w.writes > 0 {
		return 0, context.Canceled
	}
	written, err := w.writer.Write(p)
	if err == nil {
		*w.writtenBytes += written
		*w.writes++
	}
	w.cancel()
	return written, err
}

func TestBinaryStableReadRejectsShortReadAndFinalCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "short.png")
	put(t, path, []byte("stable bytes"))
	file, _, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	token := file.Token()
	shortHook := binaryStableReadHook{reader: func(*os.File) io.Reader {
		return &shortReader{data: []byte("stable by")}
	}}
	_, err = readBinarySnapshotHook(context.Background(), token, shortHook)
	if !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}

	readErr := errors.New("read failure")
	failHook := binaryStableReadHook{reader: func(*os.File) io.Reader {
		return errorReader{err: readErr}
	}}
	if _, err = readBinarySnapshotHook(context.Background(), token, failHook); !errors.Is(err, readErr) {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelHook := binaryStableReadHook{finish: func(*os.File) error {
		cancel()
		return nil
	}}
	_, err = readBinarySnapshotHook(ctx, token, cancelHook)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type shortReader struct {
	data []byte
	size int64
}

func (r *shortReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestBinarySameSizeRestoredMtimeReplacementIsRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "same.png")
	put(t, path, []byte("first"))
	file, snapshot, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	put(t, path, []byte("third"))
	if err = os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) ||
		!os.SameFile(before, after) {
		t.Fatal(after.Size(), before.Size(), after.ModTime(), before.ModTime(), os.SameFile(before, after))
	}
	if _, err = file.Snapshot(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	var retained bytes.Buffer
	if _, err = snapshot.WriteTo(&retained); err != nil || retained.String() != "first" {
		t.Fatal(err, retained.String())
	}
}

func TestBinaryReplaceByRenameChangesIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rename.png")
	put(t, path, []byte("first"))
	coarseStamp := time.Now().Truncate(time.Second)
	if err := os.Chtimes(path, coarseStamp, coarseStamp); err != nil {
		t.Fatal(err)
	}
	file, _, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	before := file.disk.info
	temp := filepath.Join(dir, "replacement.tmp")
	put(t, temp, []byte("first"))
	if err = os.Rename(temp, path); err != nil {
		t.Fatal(err)
	}
	if err = os.Chtimes(path, coarseStamp, coarseStamp); err != nil {
		t.Fatal(err)
	}
	afterFile, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	after, err := afterFile.Stat()
	closeErr := afterFile.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) ||
		!bytes.Equal(read(t, path), []byte("first")) || os.SameFile(before, after) {
		t.Fatal(after.Size(), before.Size(), after.ModTime(), before.ModTime(), os.SameFile(before, after))
	}
	check, err := file.Token().CheckDisk(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !check.Changed {
		t.Fatal("replace-by-rename was not detected")
	}
	if _, err = file.Snapshot(context.Background()); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
}

func TestBinaryEmptyCopyAndDeletedSourceHeldSnapshot(t *testing.T) {
	dir := t.TempDir()
	emptySource := filepath.Join(dir, "empty.png")
	put(t, emptySource, nil)
	file, emptySnapshot, err := OpenBinary(context.Background(), emptySource, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	emptyDestination := filepath.Join(dir, "empty-copy.png")
	if result, err := file.SaveAs(context.Background(), emptyDestination, emptySnapshot); err != nil ||
		!result.Committed || !result.Verified || !result.Adopted || result.NoRewrite {
		t.Fatal(result, err)
	}
	if info, err := os.Stat(emptyDestination); err != nil || info.Size() != 0 {
		t.Fatal(err, info.Size())
	}

	source := filepath.Join(dir, "held.png")
	put(t, source, []byte("held bytes"))
	file, heldSnapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(source); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, "held-copy.png")
	if result, err := file.SaveAs(context.Background(), destination, heldSnapshot); err != nil ||
		!result.Committed || !result.Verified || !result.Adopted {
		t.Fatal(result, err)
	}
	if !bytes.Equal(read(t, destination), []byte("held bytes")) {
		t.Fatal("held snapshot copy changed bytes")
	}
}

func TestBinaryCrossAuthoritySnapshotCannotBeAdopted(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "same.png")
	put(t, source, []byte("same bytes"))
	first, _, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	second, foreign, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, "foreign.png")
	if result, err := first.SaveAs(context.Background(), destination, foreign); !errors.Is(err, ErrStaleBinarySnapshot) ||
		result.Committed || result.Verified || result.Adopted {
		t.Fatal(result, err)
	}
	if _, err = os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if !second.SameBaseline(foreign.Token()) {
		t.Fatal("second authority rejected its own snapshot")
	}
}

type binaryCopyCompletion struct {
	result  BinarySaveResult
	receipt BinaryCopyReceipt
	err     error
}

func TestBinaryCopyWorkerPublicationAndReboundSnapshot(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "source.png"), filepath.Join(dir, "copy.png")
	original := bytes.Repeat([]byte{0, 1, 0xff, 'x'}, 20<<10)
	put(t, source, original)
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	originalToken := file.Token()
	completed := make(chan binaryCopyCompletion, 1)
	go func() {
		result, receipt, err := PublishBinaryCopy(context.Background(), destination, snapshot)
		completed <- binaryCopyCompletion{result, receipt, err}
	}()
	concurrentRead, readErr := io.ReadAll(snapshot.Reader())
	publication := <-completed
	if readErr != nil || !bytes.Equal(concurrentRead, original) {
		t.Fatal(readErr, "concurrent snapshot read changed bytes")
	}
	if publication.err != nil || !publication.result.Committed || !publication.result.Verified ||
		publication.result.Adopted || publication.result.NoRewrite {
		t.Fatal(publication.result, publication.err)
	}
	if publication.receipt.Path() != destination || publication.receipt.Result() != publication.result ||
		!file.SameBaseline(originalToken) {
		t.Fatal("worker changed the owner or lost its publication outcome")
	}
	if !bytes.Equal(read(t, source), original) || !bytes.Equal(read(t, destination), original) {
		t.Fatal("publication changed the accepted bytes")
	}
	result, rebound, err := file.AdoptCopy(publication.receipt, snapshot)
	if err != nil || !result.Committed || !result.Verified || !result.Adopted || result.NoRewrite || rebound == nil {
		t.Fatal(result, err)
	}
	if rebound == snapshot || &rebound.data[0] != &snapshot.data[0] || !file.SameBaseline(rebound.Token()) ||
		file.SameBaseline(originalToken) || !snapshot.Token().same(originalToken) || file.Revision() != 2 ||
		file.Path() != destination || publication.receipt.Result().Adopted {
		t.Fatal("adoption did not rebind the descriptor while retaining immutable accepted storage")
	}
	retained, err := io.ReadAll(snapshot.Reader())
	if err != nil || !bytes.Equal(retained, original) {
		t.Fatal(err, "old snapshot changed after adoption")
	}
	next := filepath.Join(dir, "next.png")
	if result, err = file.SaveAs(context.Background(), next, rebound); err != nil || !result.Adopted ||
		!result.Verified || !result.Committed || !bytes.Equal(read(t, next), original) || file.Revision() != 3 {
		t.Fatal(result, err)
	}
	current := file.Token()
	if result, rebound, err = file.AdoptCopy(publication.receipt, snapshot); !errors.Is(err, ErrStaleBinarySnapshot) ||
		result != publication.result || rebound != nil || !file.SameBaseline(current) {
		t.Fatal(result, err, "replayed publication changed its newer owner")
	}
}

func TestBinaryCopyRejectsZeroForeignAndMismatchedReceipt(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "same.jpg"), filepath.Join(dir, "copy.jpg")
	put(t, source, []byte("same source, distinct authorities"))
	owner, snapshot, err := OpenBinary(context.Background(), source, BinaryKindJPEG)
	if err != nil {
		t.Fatal(err)
	}
	foreign, foreignSnapshot, err := OpenBinary(context.Background(), source, BinaryKindJPEG)
	if err != nil {
		t.Fatal(err)
	}
	result, receipt, err := PublishBinaryCopy(context.Background(), destination, snapshot)
	if err != nil || !result.Committed || !result.Verified {
		t.Fatal(result, err)
	}
	for _, test := range []struct {
		name     string
		owner    *BinaryFile
		receipt  BinaryCopyReceipt
		snapshot *BinarySnapshot
		wantErr  error
	}{
		{"zero receipt", owner, BinaryCopyReceipt{}, snapshot, ErrInvalidBinaryCopyReceipt},
		{"foreign owner", foreign, receipt, snapshot, ErrStaleBinarySnapshot},
		{"foreign snapshot", owner, receipt, foreignSnapshot, ErrStaleBinarySnapshot},
		{"nil snapshot", owner, receipt, nil, ErrStaleBinarySnapshot},
		{"zero snapshot", owner, receipt, &BinarySnapshot{}, ErrStaleBinarySnapshot},
		{"nil owner", nil, receipt, snapshot, ErrStaleBinarySnapshot},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := test.owner.Token()
			got, rebound, err := test.owner.AdoptCopy(test.receipt, test.snapshot)
			if !errors.Is(err, test.wantErr) || rebound != nil || got != test.receipt.Result() {
				t.Fatal(got, err)
			}
			if test.owner != nil && !test.owner.SameBaseline(before) {
				t.Fatal("rejected receipt changed the owner's baseline")
			}
		})
	}
	if result, _, err = owner.AdoptCopy(receipt, snapshot); err != nil || !result.Adopted ||
		!bytes.Equal(read(t, destination), []byte("same source, distinct authorities")) {
		t.Fatal(result, err, "rejection consumed a valid receipt or removed its destination")
	}
	for _, invalid := range []*BinarySnapshot{nil, {}} {
		path := filepath.Join(dir, "invalid.png")
		result, receipt, err := PublishBinaryCopy(context.Background(), path, invalid)
		if !errors.Is(err, ErrNoBinaryBaseline) || result != (BinarySaveResult{}) || receipt.Result() != result {
			t.Fatal(result, err)
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err, "invalid snapshot published a destination")
		}
	}
}

func TestBinaryCopyPendingReceiptStaleAfterAnotherAdoption(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	put(t, source, []byte("two completed destinations"))
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "first.png")
	_, firstReceipt, err := PublishBinaryCopy(context.Background(), first, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(dir, "second.png")
	secondResult, secondReceipt, err := PublishBinaryCopy(context.Background(), second, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	_, rebound, err := file.AdoptCopy(firstReceipt, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if result, rejected, err := file.AdoptCopy(secondReceipt, snapshot); !errors.Is(err, ErrStaleBinarySnapshot) ||
		rejected != nil || result != secondResult || !result.Committed || !result.Verified ||
		!file.SameBaseline(rebound.Token()) || string(read(t, second)) != "two completed destinations" {
		t.Fatal(result, err, "stale success was hidden or changed the current baseline")
	}
}

func TestBinaryCopySamePathNoRewriteAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.png")
	put(t, path, []byte("unchanged baseline"))
	file, snapshot, err := OpenBinary(context.Background(), path, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	before := file.Token() // OpenBinary captured FileInfo from an open handle.
	_, older, err := PublishBinaryCopy(context.Background(), path, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	result, newer, err := PublishBinaryCopy(context.Background(), path, snapshot)
	if err != nil || result.Committed || result.Adopted || !result.Verified || !result.NoRewrite {
		t.Fatal(result, err)
	}
	currentFile, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	after, statErr := currentFile.Stat()
	closeErr := currentFile.Close()
	if statErr != nil || closeErr != nil {
		t.Fatal(statErr, closeErr)
	}
	if !os.SameFile(before.info, after) || !before.info.ModTime().Equal(after.ModTime()) {
		t.Fatal("same-path verification rewrote the file")
	}
	if adopted, rebound, err := file.AdoptCopy(newer, snapshot); err != nil || adopted != result ||
		rebound != snapshot || !file.SameBaseline(before) || file.Revision() != 1 {
		t.Fatal(adopted, err)
	}
	for _, consumed := range []BinaryCopyReceipt{newer, older} {
		if got, rebound, err := file.AdoptCopy(consumed, snapshot); !errors.Is(err, ErrReplayedBinaryCopyReceipt) ||
			got != result || rebound != nil || !file.SameBaseline(before) {
			t.Fatal(got, err, "replayed or superseded same-path receipt was accepted")
		}
	}
	_, fresh, err := PublishBinaryCopy(context.Background(), path, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if got, rebound, err := file.AdoptCopy(fresh, snapshot); err != nil || got != result || rebound != snapshot ||
		!file.SameBaseline(before) {
		t.Fatal(got, err, "fresh verification could not be accepted after replay rejection")
	}
	put(t, path, []byte("external change"))
	failed, receipt, err := PublishBinaryCopy(context.Background(), path, snapshot)
	if !errors.Is(err, ErrConflict) || failed.Verified || failed.Committed || failed.NoRewrite {
		t.Fatal(failed, err)
	}
	if got, rebound, err := file.AdoptCopy(receipt, snapshot); !errors.Is(err, ErrInvalidBinaryCopyReceipt) ||
		got != failed || rebound != nil || !file.SameBaseline(before) {
		t.Fatal(got, err)
	}
}

func TestBinaryCopyOwnerReloadsWhileWorkerPublishes(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "source.png"), filepath.Join(dir, "copy.png")
	put(t, source, []byte("accepted before reload"))
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	ready, proceed := make(chan struct{}), make(chan struct{})
	completed := make(chan binaryCopyCompletion, 1)
	go func() {
		hooks := binarySaveHooks{beforePublish: func() error {
			close(ready)
			<-proceed
			return nil
		}}
		result, receipt, err := publishBinaryCopy(context.Background(), destination, snapshot, hooks)
		completed <- binaryCopyCompletion{result, receipt, err}
	}()
	select {
	case <-ready:
	case early := <-completed:
		t.Fatal("worker failed before publication barrier", early.err)
	}
	writeErr := os.WriteFile(source, []byte("new owner bytes"), 0600)
	current, reloadErr := file.Reload(context.Background())
	close(proceed)
	publication := <-completed
	if writeErr != nil || reloadErr != nil || publication.err != nil {
		t.Fatal(writeErr, reloadErr, publication.err)
	}
	if !publication.result.Committed || !publication.result.Verified || publication.result.Adopted ||
		string(read(t, destination)) != "accepted before reload" || !file.SameBaseline(current.Token()) {
		t.Fatal(publication.result, "worker used mutable owner state")
	}
	if got, rebound, err := file.AdoptCopy(publication.receipt, snapshot); !errors.Is(err, ErrStaleBinarySnapshot) ||
		got != publication.result || rebound != nil || !file.SameBaseline(current.Token()) ||
		string(read(t, destination)) != "accepted before reload" {
		t.Fatal(got, err, "reload did not prevent stale adoption or removed a completed copy")
	}
}

func TestBinaryCopyFailureReceiptsCannotAdopt(t *testing.T) {
	for _, phase := range []string{"cancel before publication", "cancel after commit", "directory sync", "changed before verify"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			source, destination := filepath.Join(dir, "source.png"), filepath.Join(dir, "copy.png")
			original := []byte("accepted bytes")
			put(t, source, original)
			file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
			if err != nil {
				t.Fatal(err)
			}
			before := file.Token()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var hooks binarySaveHooks
			wantErr := error(context.Canceled)
			wantCommitted := true
			wantBytes := original
			switch phase {
			case "cancel before publication":
				hooks.beforePublish = func() error { cancel(); return nil }
				wantCommitted = false
			case "cancel after commit":
				hooks.syncDir = func(string) error { cancel(); return nil }
			case "directory sync":
				wantErr = os.ErrDeadlineExceeded
				hooks.syncDir = func(string) error { return wantErr }
			case "changed before verify":
				wantErr = ErrConflict
				wantBytes = []byte("external bytes")
				hooks.verify = func(ctx context.Context, path string) (identity, error) {
					if err := os.WriteFile(path, wantBytes, 0600); err != nil {
						return identity{}, err
					}
					return readStableDigestContext(ctx, path, MaxBinaryBytes)
				}
			}
			result, receipt, err := publishBinaryCopy(ctx, destination, snapshot, hooks)
			if !errors.Is(err, wantErr) || result.Committed != wantCommitted || result.Verified ||
				result.Adopted || result.NoRewrite || receipt.Path() != destination || receipt.Result() != result {
				t.Fatal(result, err)
			}
			if got, rebound, err := file.AdoptCopy(receipt, snapshot); !errors.Is(err, ErrInvalidBinaryCopyReceipt) ||
				got != result || rebound != nil || !file.SameBaseline(before) {
				t.Fatal(got, err, "failed receipt changed its owner or hid commit status")
			}
			if wantCommitted {
				if !bytes.Equal(read(t, destination), wantBytes) {
					t.Fatal("post-commit failure changed or removed the destination")
				}
			} else if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err, "canceled publication created a destination")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".levnote-save-") {
					t.Fatal("publication left a temporary file")
				}
			}
		})
	}
}

func TestBinaryCopyHeldSnapshotAfterSourceDeletion(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "source.png"), filepath.Join(dir, "copy.png")
	put(t, source, []byte("held immutable data"))
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	result, receipt, err := PublishBinaryCopy(context.Background(), destination, snapshot)
	if err != nil || !result.Committed || !result.Verified || !file.SameBaseline(snapshot.Token()) {
		t.Fatal(result, err)
	}
	if result, rebound, err := file.AdoptCopy(receipt, snapshot); err != nil || !result.Adopted || rebound == nil ||
		!file.SameBaseline(rebound.Token()) || string(read(t, destination)) != "held immutable data" {
		t.Fatal(result, err)
	}
}

func TestBinaryCopyReceiptRetainsExclusiveRaceOutcome(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "source.png"), filepath.Join(dir, "race.png")
	put(t, source, []byte("candidate"))
	file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	hooks := binarySaveHooks{beforePublish: func() error {
		return os.WriteFile(destination, []byte("exclusive winner"), 0600)
	}}
	result, receipt, err := publishBinaryCopy(context.Background(), destination, snapshot, hooks)
	if !errors.Is(err, ErrExists) || result != (BinarySaveResult{}) || receipt.Result() != result ||
		string(read(t, destination)) != "exclusive winner" {
		t.Fatal(result, err)
	}
	if got, rebound, err := file.AdoptCopy(receipt, snapshot); !errors.Is(err, ErrInvalidBinaryCopyReceipt) ||
		got != result || rebound != nil || !file.SameBaseline(snapshot.Token()) {
		t.Fatal(got, err)
	}
}

func TestBinaryCopyAdoptionObservesPublicationTimeIdentity(t *testing.T) {
	for _, remove := range []bool{false, true} {
		name := "external change"
		if remove {
			name = "external deletion"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			source, destination := filepath.Join(dir, "source.png"), filepath.Join(dir, "copy.png")
			put(t, source, []byte("publication bytes"))
			file, snapshot, err := OpenBinary(context.Background(), source, BinaryKindPNG)
			if err != nil {
				t.Fatal(err)
			}
			_, receipt, err := PublishBinaryCopy(context.Background(), destination, snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if remove {
				if err := os.Remove(destination); err != nil {
					t.Fatal(err)
				}
			} else {
				put(t, destination, []byte("external replacement"))
			}
			result, rebound, err := file.AdoptCopy(receipt, snapshot)
			if err != nil || !result.Adopted || !result.Verified || !result.Committed || rebound == nil {
				t.Fatal(result, err, "adoption tried to reread or reinterpret completed publication")
			}
			data, readErr := io.ReadAll(rebound.Reader())
			if readErr != nil || string(data) != "publication bytes" {
				t.Fatal(readErr, "adoption substituted external bytes for held data")
			}
			check, err := rebound.Token().CheckDisk(context.Background())
			if err != nil || !check.Changed || check.Missing != remove || check.Invalid {
				t.Fatal(check, err, "publication identity lost subsequent conflict observation")
			}
		})
	}
}
