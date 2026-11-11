# .29 current product documentation accepted

Accepted2026-11-10. Root read actual current entries, implementation/evidence
wording, saved report, original33/37 check and narrow3/4 correction, final
identities, and file-scoped diffs against backed-up VALIDATION/ARCHITECTURE/
ROADMAP. All six final doc hashes match the owner's external record. Root did
not execute a docs verifier or product checks. A mistaken directory-wide read-only
diff encountered an unrelated long-path access error; it is not verification
evidence. The subsequent named-file diffs provide the review evidence.

Both executed check results remain exit1; this acceptance does not rewrite
them as passing. First script had heading whitespace, slash punctuation,
inline-code link false positives and historical entry relabel mismatches.
The narrow correction passed the first three, including584actual relative
targets after excluding9inline-code examples; its remaining predicate failed
before comparison because it overlooked the blank line before 历史入口.
Root's actual VALIDATION file diff shows only the new .29 section plus .28
heading/entry current-to-historical labels, with the rest of that historical
file unchanged. No third verifier is necessary to establish this visible
documentation-only fact. Original scripts/streams/results remain preserved.

Current entries and both formerly stale ROADMAP rows now point to accepted
.29. Composition endpoint behavior/irregular-range limits, .28 size-refusal
barriers and source-test versus physical/whole-process claims are accurately
distinguished. Historical .28/.27 facts and failures are retained. Token
ownership stays in architecture; current usage does not claim picker shipping.

Final report7841B SHA256
D0AB9C7720D8436720BB0FDF4BC3B8D7E4B12A25952A70E3301EC3963B325F78.
External final-identities6602B SHA256
A11375E0CA621A3A2CD68F6171A93BC87F25E3A98FBC554CBF602731C9F74F3C.
Six document identities:
- README.md21722B CC55C274A60FF2D178387DC5939D08ADDAB0D99ED5AD1F6E8165D8AB8599427C
- ROADMAP.md125922B D6C3196A913D33E3B0ABC1DF0227064E7639BBF5710ED5975C8B08733398DC3B
- CHANGELOG.md157759B 5D15C35D511CBD1A81AA9A06D9F0E7E2CFEEA38F525AC2881CD23FABA4E683D7
- native/README.md74225B D73F310BF6089D22331D930EC0FDC636924607B026BD6687F312D89A29493CCA
- native/ARCHITECTURE.md73027B C841FEAA490AC15A236783A764D35EB552F9EFB3776344BD2F9F2978CEC04089
- native/VALIDATION.md178532B 3CD044E232F3E49A3F3BA15C26E190A060BE5DE852882FB52F4475522F4990CA

Docs ownership released. Activate open-document-picker-release30-preparation-task.md
for the delivery owner in new helpers/portable README only. No build/host/context
until picker source acceptance; source implementation and review continue.
