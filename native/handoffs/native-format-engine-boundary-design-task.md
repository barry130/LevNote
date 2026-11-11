# Next format-engine boundary — finite design only

Status: DESIGN ONLY2026-11-09. The user wants format support to become plugins,
not only extra filename extensions or syntax configurations. Current alignment
consumer/package work stays with Astra and all product inputs must remain
untouched. Root coordinates; GLM may write only
native/handoffs/native-format-engine-boundary-design.md. No implementation,
test/build/process/network execution or delegation. Keep context below200K.

Use actual current native/ARCHITECTURE.md, formats/API.md, ui/editor/contract.go,
core/plugins, core/engineipc, core/engineruntime and modules composition to make
one bounded next architectural proposal. Installed JSON schema1/2 source
highlight/outline is already implemented. The only separate engine currently is
fixed Markdown HTML export; parsing/editing/preview remain compiled native code.
Do not recommend redoing schema2, a cosmetic plugin label, or extension aliases
as completion of the user's engine goal.

Decide one finite implementation slice that materially establishes independently
packaged project-authored format engines while retaining native UI, the single
authoritative text buffer, undo/file safety, lazy loading and explicit bounds.
Consider whether a repository-owned engine catalog/capability descriptor with a
second concrete export engine is a useful tested bridge, versus introducing a
bounded read-only parse/outline operation for the existing Markdown worker. Give
one recommendation using current code seams and a concrete user-visible result.
Do not expand into all rich views, arbitrary installed executables, dynamic DLLs,
generic RPC frameworks, sandbox claims or cross-platform work in this slice.

Specify only what root needs to decide: minimal owned files and API/DTO changes,
identity/version and cancellation lifetime, loading/unloading/bytes ownership,
source fidelity, per-operation resource limits, missing/disabled/failing-engine
behavior and meaningful finite acceptance. Separate installed data identity,
format capability identity and executable identity. Explain exactly what will
still be compiled into the app afterward and the next missing contract, without
a long speculative backlog or pretending export-only means editing is modular.

Use at most1500words and a few verified references. Read only the files needed
to ground the proposal; send a quick decision if a crucial conflict appears.
Finish within one bounded review turn, roughly10minutes; uncertainty may be
stated rather than exhaustively explored. The design does not authorize changes
to any current package/evidence/source docs. Root will select and stage work
after alignment delivery.
