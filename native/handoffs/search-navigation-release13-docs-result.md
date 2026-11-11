# Search navigation and Find layout: `.13` docs result

2026-11-09. The six-file documentation follow-up is complete and frozen for
root review. The documents describe the accepted `.13` source and delivery
evidence, retain `.12`/`.11` as history, and explicitly exclude the separate,
still-evolving Find keyboard work.

## Changed files

- [README.md](../../README.md)
- [ROADMAP.md](../../ROADMAP.md)
- [CHANGELOG.md](../../CHANGELOG.md)
- [native/README.md](../README.md)
- [native/ARCHITECTURE.md](../ARCHITECTURE.md)
- [native/VALIDATION.md](../VALIDATION.md)

The first status paragraph of
[search-navigation-release13-delivery.md](search-navigation-release13-delivery.md)
now points to the
[root acceptance](search-navigation-release13-root-acceptance.md). Its candidate
facts, evidence and limits are otherwise unchanged.

## Documented behavior

The current runnable path is
`native/bin/levnote-native-worker-20261109.13/levnote-native.exe`. The documents
describe:

- adjacent previous/next correctness using ordered UTF-16 selection bounds,
  nearest-result selection and wraparound;
- one active-document offset cache with exact owner/version/query identity,
  Flush-before-find, committed-transition invalidation and fresh replacement
  authority;
- minimum 760x480 Find-bar wrapping through exactly two existing-row `Wrap`
  calls, with unchanged actions, Tab order and search semantics;
- retained `.12` `Ctrl+G` navigation and `.11` pointer-lifecycle fixes.

The function-only benchmark is labeled as such: warm lookup reuses offsets, the
cold path still scans the complete text, and no whole-app speed or resident
memory claim is made. The two layout captures are scoped to tester geometry and
do not become physical OS IME, visible-interaction or resource evidence.

Root review corrected the exact-match wording to the requested previous/next
operation direction, which applies to forward and reversed selections. The
validation record now also separates PNG geometry from the MyGo Tester event
checks that establish resize, input, options, navigation, replace/close actions
and Tab traversal.

## Evidence links

- [Search navigation source acceptance](search-navigation-root-acceptance.md)
- [Search navigation implementation result](search-navigation-result.md)
- [Search navigation independent review](search-navigation-independent-review.md)
- [Find-bar layout source acceptance](findbar-layout-root-acceptance.md)
- [Find-bar layout result](findbar-layout-result.md)
- [`.13` package acceptance](search-navigation-release13-root-acceptance.md)
- [`.13` delivery](search-navigation-release13-delivery.md)
- [`.13` validator](../artifacts/acceptance/search-navigation-release13-delivery-20261109-210329/validator.json)

## Read-only checks

The six documents contained 320 local Markdown targets after the edits; all 320
resolved to existing paths (0 missing). HTTP links were outside the local-path
check. The documented executable exists with 14,031,872 bytes and SHA-256
`4B86002034A493789A1747E343542FD5A94BCD9DCEB5060662A5DD55DCF4150D`.
The adjacent 4,137-byte release manifest also exists with SHA-256
`0F1825348BD3EEFDB8264079405DBA23019692D22EA7962098B3A4FC4470D5FB`.

No broken local links required correction after the first pass. No tests,
builds, launches, package writes, product-source edits or evidence edits were
performed. `.12`, `.11` and all earlier release/evidence records remain intact.
