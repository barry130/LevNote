// Package code provides token-based lexical highlighting with no parsing or execution.
package code

type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
	LanguageJavaScript Language = "javascript"
	LanguagePython     Language = "python"
	LanguageJSON       Language = "json"
	LanguageRust       Language = "rust"
	LanguageXML        Language = "xml"
)

type Token string

const (
	TokenKeyword Token = "keyword"
	TokenString  Token = "string"
	TokenNumber  Token = "number"
	TokenComment Token = "comment"
	TokenTag     Token = "tag"
	TokenName    Token = "name"
	TokenPunct   Token = "punct"
	TokenText    Token = "text"
)

type Span struct {
	Start int
	End   int
	Token Token
	Kind  string
}

type Warning struct {
	Code    string
	Message string
	Range   SourceRange
}

type SourceRange struct {
	Start int
	End   int
}

type Result struct {
	Spans    []Span
	Warnings []Warning
}

// DeclarativeSyntax is bounded data supplied by a plugin manifest. It styles
// text only; it never supplies functions or commands to execute.
type DeclarativeSyntax struct {
	LineComment string
	BlockStart  string
	BlockEnd    string
	Keywords    []string
}

const (
	maxHighlightBytes = 2 << 20
	maxHighlightWork  = 8_000_000
)
