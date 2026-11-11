// Package engineruntime runs fixed, repository-owned one-shot export engines.
// It owns process lifetime and bounded transport, never document publication.
package engineruntime

import (
	"io"
	"unicode/utf8"

	"levnote.local/native/core/engineipc"
)

// Source is captured on the UI thread. Data must be immutable and must not
// block independently of writing to the supplied destination. Runner.Export
// copies Page into operation-owned storage during admission; later caller
// mutation of the original RequestPage cannot alter the running request.
type Source struct {
	Size int64
	Data io.WriterTo
	Page *engineipc.RequestPage
}

type EngineInfo struct {
	ID            engineipc.EngineID
	Version       string
	SHA256        string
	Source        PackageSource
	PackageDigest string
}

type Result struct {
	Payload  []byte
	Warnings []engineipc.Warning
	Engine   EngineInfo
}

type FailureKind string

const (
	FailureBusy             FailureKind = "busy"
	FailureClosed           FailureKind = "closed"
	FailureUnavailable      FailureKind = "unavailable"
	FailureIntegrity        FailureKind = "integrity"
	FailureSourceLimit      FailureKind = "source-limit"
	FailureSourceSize       FailureKind = "source-size"
	FailureProtocol         FailureKind = "protocol"
	FailureStdoutLimit      FailureKind = "stdout-limit"
	FailureStderrLimit      FailureKind = "stderr-limit"
	FailureWorkerExit       FailureKind = "worker-exit"
	FailureUnexpectedStderr FailureKind = "unexpected-stderr"
	FailureEngine           FailureKind = "engine"
	FailureCanceled         FailureKind = "canceled"
	FailureTimeout          FailureKind = "timeout"
	FailureCleanup          FailureKind = "cleanup"
)

// Failure deliberately contains no document bytes or raw child stderr.
type Failure struct {
	Kind   FailureKind
	Code   string
	Detail string
	cause  error
}

func (e *Failure) Error() string { return "export engine " + string(e.Kind) + ": " + e.Detail }
func (e *Failure) Unwrap() error { return e.cause }

func failure(kind FailureKind, detail string, cause error) *Failure {
	// Production callers use constant descriptions or already validated worker
	// messages; retain a defensive byte bound without splitting UTF-8.
	if len(detail) > 1024 {
		detail = detail[:1024]
		for !utf8.ValidString(detail) {
			detail = detail[:len(detail)-1]
		}
	}
	return &Failure{Kind: kind, Detail: detail, cause: cause}
}
