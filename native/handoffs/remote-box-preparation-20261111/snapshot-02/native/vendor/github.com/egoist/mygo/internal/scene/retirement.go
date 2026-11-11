package scene

import "sync/atomic"

// RetirementEpoch is shared notification, not a consumable queue. Every
// renderer observes it independently. Image mutation remains main-thread only,
// after any synchronous CPU rendering workers have joined.
func RetirementEpoch() uint64 { return imageRetirements.Load() }

var imageRetirements atomic.Uint64

// Retired includes nil, which every image lookup treats as transparent.
func (m *Image) Retired() bool { return m == nil || m.retired }

// Retire drops owned pixels and restoration closures, once. Old display lists
// keep a small tombstone with its original identity and geometry. Canvases are
// shared retained scenes, not owned bitmaps, and cannot be retired this way.
func (m *Image) Retire() bool {
	if m.Retired() {
		return true
	}
	if m.Scene != nil {
		return false
	}
	m.retired = true
	m.version++
	m.Pix, m.Restore, m.Dropped = nil, nil, nil
	imageRetirements.Add(1)
	return true
}
