package document

import (
	"context"
	"errors"
	"io"
	"os"
	"sync/atomic"
)

var (
	ErrInvalidBinaryCopyReceipt  = errors.New("binary copy receipt is not verified")
	ErrReplayedBinaryCopyReceipt = errors.New("binary copy receipt was already adopted or superseded")
)

var lastBinaryCopy atomic.Uint64

// BinaryCopyReceipt is an immutable publication record. It holds only source
// authority, verified destination identity and operation status, never a live
// BinaryFile, snapshot, byte buffer, reader or open handle. Its private fields
// cannot be used to manufacture adoption. The zero value cannot be adopted.
//
// Verification describes the destination at publication time. External writers
// can change it afterward; adoption is not filesystem CAS and does not reread
// it. Observe later changes with the adopted token's CheckDisk.
type BinaryCopyReceipt struct {
	source      BinaryToken
	destination string
	disk        identity
	result      BinarySaveResult
	serial      uint64
}

// Path is the resolved destination, including for a post-publication failure.
func (r BinaryCopyReceipt) Path() string { return r.destination }

// Result preserves the worker's outcome, even when owner adoption is refused.
// Adopted is always false here: accepting a receipt does not mutate the receipt.
func (r BinaryCopyReceipt) Result() BinarySaveResult { return r.result }

// PublishBinaryCopy exclusively writes snapshot's exact accepted bytes to path
// and verifies with a bounded streaming digest. It may run on a worker: it
// receives and touches no mutable BinaryFile. The snapshot may remain usable
// even after its source is changed/deleted or its owner reloads/closes.
//
// Committed remains true on cancellation or verification failure after atomic
// publication; the destination is never removed to conceal that success. The
// host must report that outcome even if the originating tab closes/changes and
// cannot adopt. Same-path publication verifies without writing or committing.
// Return the receipt and original snapshot to the owner's AdoptCopy method.
func PublishBinaryCopy(ctx context.Context, path string, snapshot *BinarySnapshot) (BinarySaveResult, BinaryCopyReceipt, error) {
	return publishBinaryCopy(ctx, path, snapshot, binarySaveHooks{})
}

func publishBinaryCopy(ctx context.Context, path string, snapshot *BinarySnapshot, hooks binarySaveHooks) (BinarySaveResult, BinaryCopyReceipt, error) {
	if err := ctx.Err(); err != nil {
		return BinarySaveResult{}, BinaryCopyReceipt{}, err
	}
	if snapshot == nil || !snapshot.token.Valid() || snapshot.token.info == nil {
		return BinarySaveResult{}, BinaryCopyReceipt{}, ErrNoBinaryBaseline
	}
	destination, err := canonical(path)
	if err != nil {
		return BinarySaveResult{}, BinaryCopyReceipt{}, err
	}
	receipt := BinaryCopyReceipt{source: snapshot.token, destination: destination, serial: lastBinaryCopy.Add(1)}
	if destination == snapshot.token.path {
		digest := hooks.digest
		if digest == nil {
			digest = readStableDigestContext
		}
		disk, err := digest(ctx, destination, MaxBinaryBytes)
		if err != nil {
			return receipt.result, receipt, err
		}
		if !sameIdentity(snapshot.token.identity(), disk) {
			return receipt.result, receipt, &Error{"save as binary", destination, ErrConflict, nil}
		}
		receipt.disk = disk
		receipt.result = BinarySaveResult{Verified: true, NoRewrite: true}
		return receipt.result, receipt, nil
	}
	if _, err = os.Lstat(destination); err == nil {
		return receipt.result, receipt, &Error{"save as binary", destination, ErrExists, nil}
	} else if !errors.Is(err, os.ErrNotExist) {
		return receipt.result, receipt, err
	}
	write := func(w io.Writer) error {
		if hooks.beforeWrite != nil {
			if err := hooks.beforeWrite(); err != nil {
				return err
			}
		}
		writer := w
		if hooks.wrapWriter != nil {
			writer = hooks.wrapWriter(w)
		}
		return writeBinarySnapshot(ctx, writer, snapshot)
	}
	publicationCheck := func() error {
		if hooks.beforePublish != nil {
			if err := hooks.beforePublish(); err != nil {
				return err
			}
		}
		return ctx.Err()
	}
	outcome, err := writeAtomicWriterOutcome(destination, 0600, true, publicationCheck, write, hooks.syncDir)
	receipt.result.Committed = outcome.Committed
	if err != nil {
		return receipt.result, receipt, err
	}
	verify := hooks.verify
	if verify == nil {
		digest := hooks.digest
		if digest == nil {
			digest = readStableDigestContext
		}
		verify = func(ctx context.Context, path string) (identity, error) {
			return digest(ctx, path, MaxBinaryBytes)
		}
	}
	disk, err := verify(ctx, destination)
	if err != nil {
		return receipt.result, receipt, &Error{"verify binary save", destination, err, nil}
	}
	if disk.info == nil || disk.hash != snapshot.token.hash || disk.info.Size() != snapshot.Size() {
		return receipt.result, receipt, &Error{"verify binary save", destination, ErrConflict, nil}
	}
	receipt.disk, receipt.result.Verified = disk, true
	return receipt.result, receipt, nil
}

// AdoptCopy accepts a verified receipt on the BinaryFile owner's goroutine.
// Both the current baseline and supplied immutable snapshot must match the
// receipt's source authority before anything changes. A failed/stale/foreign/
// replayed receipt returns its truthful publication result and no snapshot;
// an already committed destination is left intact.
//
// A new destination increments Revision and returns a fresh snapshot descriptor
// sharing the accepted bytes with a new token. The old snapshot stays immutable
// and stale. Same-path verification returns the supplied snapshot unchanged,
// without revision change or Adopted=true. Successful handling consumes the
// serial on this owner; older receipts on that baseline are superseded, keeping
// replay state bounded. No filesystem IO or byte-sized allocation occurs here.
func (f *BinaryFile) AdoptCopy(receipt BinaryCopyReceipt, snapshot *BinarySnapshot) (BinarySaveResult, *BinarySnapshot, error) {
	result := receipt.result
	if receipt.serial == 0 || !receipt.source.Valid() || receipt.destination == "" ||
		!result.Verified || result.Adopted || result.Committed == result.NoRewrite ||
		receipt.disk.info == nil || receipt.disk.hash != receipt.source.hash ||
		receipt.disk.info.Size() != receipt.source.Size() {
		return result, nil, &Error{"adopt binary copy", receipt.destination, ErrInvalidBinaryCopyReceipt, nil}
	}
	if !f.SameBaseline(receipt.source) || snapshot == nil || !snapshot.token.same(receipt.source) {
		return result, nil, &Error{"adopt binary copy", receipt.destination, ErrStaleBinarySnapshot, nil}
	}
	if receipt.serial <= f.lastCopy {
		return result, nil, &Error{"adopt binary copy", receipt.destination, ErrReplayedBinaryCopyReceipt, nil}
	}
	if result.NoRewrite {
		if receipt.destination != f.path || !sameIdentity(f.disk, receipt.disk) {
			return result, nil, &Error{"adopt binary copy", receipt.destination, ErrInvalidBinaryCopyReceipt, nil}
		}
		f.lastCopy = receipt.serial
		return result, snapshot, nil
	}
	if receipt.destination == f.path {
		return result, nil, &Error{"adopt binary copy", receipt.destination, ErrInvalidBinaryCopyReceipt, nil}
	}
	f.path, f.disk = receipt.destination, receipt.disk
	f.readOnly = receipt.disk.info.Mode().Perm()&0200 == 0
	f.revision++
	f.lastCopy = receipt.serial
	result.Adopted = true
	return result, &BinarySnapshot{data: snapshot.data, token: f.token()}, nil
}
