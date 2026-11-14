package document

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentPublicationFailureEncodingAndSaveMatrix(t *testing.T) {
	for _, enc := range []Encoding{UTF8, UTF16LE} {
		for _, as := range []bool{false, true} {
			for _, verify := range []bool{false, true} {
				name := string(enc) + "/save"
				if as { name += "as" }
				if verify { name += "/verify" } else { name += "/sync" }
				t.Run(name, func(t *testing.T) {
					root := t.TempDir()
					source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "copy.txt")
					original, err := encode("old", enc, true)
					if err != nil { t.Fatal(err) }
					put(t, source, original)
					doc, err := Open(source)
					if err != nil { t.Fatal(err) }
					baseline := doc.DiskToken()
					buffer := doc.Buffer
					if err := buffer.ReplaceUTF16(0, 3, "甲🙂\r\nZ"); err != nil { t.Fatal(err) }
					draft, err := encode(buffer.String(), enc, true)
					if err != nil { t.Fatal(err) }
					version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
					if !as { target = source }
					sync := atomicSyncDirectory
					defer func() { atomicSyncDirectory = sync }()
					cause := &os.PathError{Op: "sync", Path: root, Err: os.ErrPermission}
					atomicSyncDirectory = func(string) error {
						if !bytes.Equal(read(t, target), draft) { t.Fatal("actual encoded publication was not exact") }
						if verify { put(t, target, []byte("external")); return nil }
						return cause
					}
					if as { err = doc.SaveAs(target) } else { err = doc.Save() }
					var publication *PublicationError
					if !errors.As(err, &publication) || publication.Path != target || publication.MetadataAdopted != (as && verify) {
						t.Fatalf("missing publication outcome: %v", err)
					}
					if verify && !errors.Is(err, ErrConflict) || !verify && !errors.Is(err, os.ErrPermission) { t.Fatalf("lost cause: %v", err) }
					if !doc.Dirty() || doc.Encoding != enc || !doc.BOM || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || doc.Buffer != buffer {
						t.Fatal("publication failure changed dirty draft/encoding/BOM/history")
					}
					if as && !bytes.Equal(read(t, source), original) { t.Fatal("SaveAs touched source") }
					if (!as || !verify) && !doc.SameBaseline(baseline) { t.Fatal("failure advanced verified baseline") }
					atomicSyncDirectory = sync
					if err := doc.SaveAs(filepath.Join(root, "retry.txt")); err != nil || doc.Dirty() || !bytes.Equal(read(t, doc.Path), draft) {
						t.Fatalf("verified fresh target retry lost exact encoded draft: %v", err)
					}
				})
			}
		}
	}
}

func TestDocumentVerifyFailureLatchAndDiskTokenRevert(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target.txt")
	doc := New("draft")
	buffer := doc.Buffer
	sync := atomicSyncDirectory
	defer func() { atomicSyncDirectory = sync }()
	atomicSyncDirectory = func(string) error { put(t, target, []byte("external")); return nil }
	if err := doc.SaveAs(target); !errors.Is(err, ErrConflict) || !doc.Dirty() || doc.DiskToken().Valid() { t.Fatalf("verify failure gave initial draft a clean baseline: %v", err) }
	atomicSyncDirectory = sync
	if err := doc.Save(); !errors.Is(err, ErrConflict) { t.Fatalf("Save accepted unverified target: %v", err) }
	if err := doc.Reload(); !errors.Is(err, ErrDirty) { t.Fatalf("Reload silently discarded dirty draft: %v", err) }
	if err := doc.Revert(); err != nil || doc.Dirty() || doc.Buffer != buffer || buffer.String() != "external" || !doc.DiskToken().Valid() { t.Fatalf("explicit Revert did not adopt verified bytes and clear latch: %v", err) }
	result, err := doc.DiskToken().CheckDisk(context.Background())
	if err != nil || result.Changed || result.Invalid || result.Missing { t.Fatalf("reverted token did not match verified disk: %+v %v", result, err) }
}

func TestDocumentPrePublicationFailuresHaveNoPublicationOutcome(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.txt")
	put(t, path, []byte("old"))
	doc, err := Open(path)
	if err != nil { t.Fatal(err) }
	if err := doc.Buffer.ReplaceUTF16(0, 3, "draft"); err != nil { t.Fatal(err) }
	baseline := doc.DiskToken()
	doc.ReadOnly = true
	err = doc.Save()
	var publication *PublicationError
	if !errors.Is(err, ErrReadOnly) || errors.As(err, &publication) { t.Fatalf("pre-publication refusal mislabeled: %v", err) }
	doc.ReadOnly = false
	existing := filepath.Join(filepath.Dir(path), "existing.txt")
	put(t, existing, []byte("external"))
	if err := doc.SaveAs(existing); !errors.Is(err, ErrExists) || errors.As(err, &publication) || string(read(t, path)) != "old" || !doc.Dirty() || !doc.SameBaseline(baseline) { t.Fatalf("existing target refusal was published: %v", err) }
}

func TestDocumentSaveAsVerifyFailurePreservesDirtyAndAdoptsTargetMetadata(t *testing.T) {
	for _, fresh := range []bool{false, true} {
		name := "existing-source"
		if fresh {
			name = "new-document"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "target.txt")
			doc := New("draft")
			if !fresh {
				put(t, source, []byte("old"))
				var err error
				doc, err = Open(source)
				if err != nil {
					t.Fatal(err)
				}
				if err := doc.Buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
					t.Fatal(err)
				}
			}
			doc.ReadOnly = true
			buffer := doc.Buffer
			version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
			sync := atomicSyncDirectory
			defer func() { atomicSyncDirectory = sync }()
			atomicSyncDirectory = func(string) error {
				if string(read(t, target)) != "draft" {
					t.Fatal("fault did not follow actual publication")
				}
				put(t, target, []byte("other"))
				return nil
			}
			err := doc.SaveAs(target)
			var publication *PublicationError
			if !errors.As(err, &publication) || publication.Path != target || !publication.MetadataAdopted || !errors.Is(err, ErrConflict) {
				t.Fatalf("verification failure lost adopted target/outcome: %v", err)
			}
			if string(read(t, target)) != "other" || doc.Path != target || doc.ReadOnly || doc.IsNew || !doc.Dirty() ||
				doc.Buffer != buffer || buffer.String() != "draft" || buffer.Version() != version ||
				buffer.Selection() != selection || buffer.History() != history {
				t.Fatal("verify failure falsely cleaned a dirty/new draft or lost target metadata")
			}
			atomicSyncDirectory = sync
			if err := doc.SaveAs(filepath.Join(root, "retry.txt")); err != nil || doc.Dirty() {
				t.Fatalf("fresh-name retry failed to establish verified clean baseline: %v", err)
			}
		})
	}
}

func TestDocumentSaveAsDisclosesPublishedDirectorySyncFailure(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "target.txt")
	put(t, source, []byte("old"))
	doc, err := Open(source)
	if err != nil {
		t.Fatal(err)
	}
	doc.ReadOnly = true
	buffer, baseline := doc.Buffer, doc.DiskToken()
	if err := buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
		t.Fatal(err)
	}
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	sync := atomicSyncDirectory
	defer func() { atomicSyncDirectory = sync }()
	cause := &os.PathError{Op: "sync", Path: root, Err: os.ErrPermission}
	atomicSyncDirectory = func(string) error { return cause }
	err = doc.SaveAs(target)
	var publication *PublicationError
	var detail *os.PathError
	if !errors.As(err, &publication) || publication.Path != target || publication.MetadataAdopted ||
		!errors.Is(err, os.ErrPermission) || !errors.As(err, &detail) || detail != cause {
		t.Fatalf("post-publication failure lost actual target/outcome/OS cause: %v; target bytes=%q", err, read(t, target))
	}
	if string(read(t, target)) != "draft" || string(read(t, source)) != "old" || doc.Path != source || !doc.ReadOnly || doc.IsNew ||
		!doc.Dirty() || !doc.SameBaseline(baseline) || doc.Buffer != buffer || buffer.Version() != version ||
		buffer.Selection() != selection || buffer.History() != history {
		t.Fatal("sync failure changed original metadata, dirty draft or real target publication")
	}
	atomicSyncDirectory = sync
	if err := doc.SaveAs(target); !errors.Is(err, ErrExists) {
		t.Fatalf("retry overwrote already published target: %v", err)
	}
	if err := doc.SaveAs(filepath.Join(root, "retry.txt")); err != nil || doc.Dirty() || doc.ReadOnly {
		t.Fatalf("fresh-name retry did not retain draft usability: %v", err)
	}
}
