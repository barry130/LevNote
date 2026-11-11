# Table-column plan review decision

2026-11-10. The bounded format/UI increment is active under its separate owner
tasks after .24 acceptance. Independent plan review is retained as advisory.

Accepted: total-width1/8 limits, original-coordinate edit list, precomputed
UTF16 offsets, descending application/selection mapping, one structural undo
group, exact unaffected-cell/neighbor/warning preservation, explicit source
fallback and no parser expansion. No-leading first deletion adds only a
structural boundary at the original indent when needed for exact preservation.

Review items2/3 are not adopted as phrased. One leading pipe plus a padding
byte does not by itself represent an inserted empty first column. The planner
must include both the leading outer boundary and new column separator. A
padding byte before a new separator after an odd-backslash cell changes that
surviving Cell.Range raw value. The active task chooses strict byte preservation
including original padding, so this case refuses rather than silently adding
padding. Minimal outer pipes are the only permitted additions outside the new
cell. Exact candidate-descriptor checks must prove that contract.

The existing parser's delimiter requires at least3hyphens. Use supported
:---/---:/:---: forms in positive fixtures; two-hyphen plan examples are invalid.
Single-pipe empty rows, leading-only nonempty one-column rows and padding-only
one-column rows are distinct scanner cases and require independent expected
outputs. No new product or evidence run is requested by this decision.
