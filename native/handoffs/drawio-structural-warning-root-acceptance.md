# Canonical drawio warning source acceptance

Accepted2026-11-10. Root read actual parse/helper/test code, final affected log,
corrected result and independent review. No root implementation/test/build/run.

Result9,778bytes SHA256
0B9DC4B7676DE115BC9CA7A0F69F1253B25365CB3644E7329EFA433CE49C9C04;
independent review7,118bytes SHA256
DE02ABF4A340CC6F29371BC60C17006F73BF4E2855AE4448F810BE4F5D6FB46F.
Six final source/test identities match reviewed result. Parse36,101bytes SHA256
92BDCA474CE49921FD88331079A186FB6CD6887F69DA8000B53C2217E1ACF8C9;
immutable SVG oracle remains
14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611.

Only fully validated first ordered canonical0/1 pair loses exactly two generic
warnings. XML-S whitespace/comments allowed, PI/non-XML whitespace/unknown attrs/
namespace/content/geometry disqualify; malformed second never removes first.
Other warning order, duplicate checks, parser budgets and source/edit/SVG bytes
retained. Vacated issue backing tail cleared. This is conservative template
support, not arbitrary layer/group interpretation. Independent review found no
production blocker. Reviewer overstates one extra-attribute fixture isolation:
that fixture also lacks parent; actual exact-attribute production check is
clear and other tests cover valid-parent variants. Not a blocker for this slice.

Final affected tests passed drawio0.725s/worker0.670s/graphview11.369s/runtime
38.040s; vet empty, author-reported exits0 from tool history. Raw logs preserved
but no exit sidecars: this provenance limit is disclosed, not independently
proved by empty logs. Focused assertion/nil/budget arithmetic failures retained.
No rerun solely to generate sidecars. Future commands should record exits then.

Current real-smoke test expects zero warnings and unchanged598byte SVG but was
gated/unexecuted. .19/.20 old two-warning evidence remains true and immutable.
No new binary/physical UI/resource claim. Accepted source may now undergo the
separately assigned ownership optimization; .21 will verify combined behavior
with one actual managed worker request. Historical source-result identities
remain recorded even when current parse.go changes in the next accepted slice.
