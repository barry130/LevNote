# .25 docs: root review corrections

2026-11-10. Actual file review found these defects; correct only the six owned
product docs/result. No runtime/tests/builds/launches. This note remains active
until the corrected result is frozen and checked.

1. README repeats the .24 image-keyboard introductory sentence consecutively.
2. VALIDATION new release-manifest SHA dropped two characters. Copy it directly
   from .25 release-manifest.json fingerprint in the accepted delivery/root
   acceptance; do not manually retype a new digest. Check64hex length/exact match.
3. VALIDATION prior .24 section lost its table, recovery/evidence/metadata-history
   paragraphs and now contains only intro. Restore the complete accepted section
   from pre-edit source/session output; change current to historical only. No
   deletion or broad rewrite of historical evidence was authorized.
4. ARCHITECTURE table capability section still says no column-insert/delete tools.
   Remove that stale current-capability statement; preserve sorting/merge/formula
   and full-GFM limitations. Shared action pipeline now includes columns too.
5. CHANGELOG should name actual visible 前插列/后插列/删此列 controls. F6 goes to
   first row action; Tab continues to column/alignment actions. Do not say F6
   directly opens column actions.
6. The preliminary result says CHANGELOG unchanged, but it now contains .25;
   generate final six hashes from final files and describe actual changes.

Perform one focused wording/link/hash check after these corrections. Preserve
the initial incomplete result as history in the result description if useful,
but no product/evidence operations. Freeze result after checks and report.
