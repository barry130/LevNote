package editor

// ExportPage is a value copy of the page selected for an export. Index is the
// format-defined page position and ID is the exact page identity, if present.
type ExportPage struct {
	Index int
	ID    string
}

// ExportToken identifies a view state that was prepared for export.
type ExportToken uint64

// ViewExportRequest ties prepared page metadata to the exact view state.
type ViewExportRequest struct {
	Page  ExportPage
	Token ExportToken
}

// ExportRequestPreparer synchronizes any host-owned cache needed to identify
// the selected export page. It must not publish source or discard drafts.
type ExportRequestPreparer interface {
	PrepareExportRequest() (ViewExportRequest, error)
}

// ExportRequestTokenProvider is side-effect-free. Hosts use it to revalidate a
// previously prepared request without synchronizing or parsing.
type ExportRequestTokenProvider interface {
	ExportRequestToken() ExportToken
}
