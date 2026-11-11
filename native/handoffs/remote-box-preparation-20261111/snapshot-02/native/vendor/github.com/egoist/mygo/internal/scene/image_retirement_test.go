package scene

import "testing"

func TestImageRetirementTombstone(t *testing.T) {
	m := NewImageRGBA(2, 3, make([]byte, 24))
	id, version, epoch := m.ID(), m.Version(), RetirementEpoch()
	m.Restore = func() []byte { t.Fatal("restored retired image"); return nil }
	m.Dropped = func() { t.Fatal("uploaded retired image") }
	if !m.Retire() || !m.Retire() || !(*Image)(nil).Retire() {
		t.Fatal("retirement must be nil/repeat safe")
	}
	m.Changed()
	m.Resize(8, 9)
	m.Uploaded()
	if !m.Retired() || m.ID() != id || m.Version() != version+1 || m.W != 2 || m.H != 3 ||
		m.Pixels() != nil || m.Pix != nil || m.Restore != nil || m.Dropped != nil || RetirementEpoch() != epoch+1 {
		t.Fatal("tombstone changed identity/geometry, retained ownership, or repeated notification")
	}
	c := NewCanvas(2, 3)
	if c.Retire() || c.Retired() || c.Version() != 1 || RetirementEpoch() != epoch+1 {
		t.Fatal("borrowed canvas was mutated")
	}
}
