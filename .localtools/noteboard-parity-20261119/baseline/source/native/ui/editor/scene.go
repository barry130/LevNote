package editor

import "levnote.local/native/core/enginescene"

// SceneCapability advertises a fixed native presentation operation. A View
// supplies neither a source document nor an executable to the host provider.
type SceneCapability struct{ ID, Format string }
type SceneRequest struct {
	Capability SceneCapability
	Epoch      uint64
}
type SceneWarning struct {
	Code, Message string
	Start, End    uint32
}
type SceneEngineInfo struct{ ID, Version, SHA256, Source, PackageDigest string }
type SceneResult struct {
	Scene    enginescene.Scene
	Warnings []SceneWarning
	Engine   SceneEngineInfo
}

// SceneRefresher is an optional UI-thread contract for readonly scene views.
// RefreshScene invalidates the old request and defers a new request until the
// next Draw/Warnings, even when source identity/version did not change. It
// preserves source and viewport state; retained content is explicitly stale
// until replacement succeeds. It does not flush or reconfigure editable views.
type SceneRefresher interface{ RefreshScene() }
