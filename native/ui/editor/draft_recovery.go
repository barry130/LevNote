package editor

// SessionDraft is an independent, immutable text snapshot. Title is a display
// label, never a path. Anchor and Head are UTF-16 offsets in Content.
type SessionDraft struct {
	Title, Content string
	Anchor, Head   int
}

// SessionDraftCapturer is an optional, read-only View capability called on the
// UI thread. It does not finish composition or modify view or document state.
// During composition it captures only previously confirmed content. A false
// result without an error means there is no independent change to recover.
// Check the byte limit before materializing text. A nonpositive maxBytes is
// never unlimited. An error leaves the host's previous recovery file intact.
type SessionDraftCapturer interface {
	CaptureSessionDraft(maxBytes int) (draft SessionDraft, ok bool, err error)
}
