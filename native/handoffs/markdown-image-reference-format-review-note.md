# Image reference parser: remaining multiline rejection hole

ACTIVE2026-11-10, DeepSeek format owner. API stays unchanged. Root found a
concrete unhandled case in final rejectedImageEnvelopeEnd: stopping at first
CR/LF allows an inner image from a complete rejected multiline envelope to
become actionable on the next line. Example:

```markdown
![bad](a
![inner](x.png)) [ordinary](ok.md)
```

The outer candidate is multiline and unsupported. It must not expose inner
InlineImage. Consume a proven matching rejected parenthesized envelope across
line endings for recovery only, under existing work/suffix budgets; resume
after its close so the ordinary link survives. Add LF/CRLF/CR meaningful
regressions, exact no-image/source/warning range and trailing-link assertions.
Do not broaden valid destination syntax. Deliberate unmatched-label recovery at
physical line remains; do not change ordinary link behavior.

Inspect similarly malformed multiline label/destination combinations for the
same path. If no closing envelope can be proven, document and test bounded
recovery semantics rather than exposing a stale partial action.

Implement only scoped format files/tests, preserve existing logs, then one
affected formats/markdown+cmd/levnote-markdown-export full test/vet after the
correction. This is a new concrete concern, not a duplicate passing-test request.
Update format result/final identities. No package/host/runtime changes elsewhere.
