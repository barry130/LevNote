# Concrete route lifecycle issue to resolve in active app slice

2026-11-09. Root read Manager.SetEnabled and app.setPluginEnabled. The manager
first SaveSettings, then swaps disabled state and calls reload, then can return
an error when enabling a target that is no longer active. App returns on any
error and skips reconfigureTabs. A genuine post-save load failure therefore
can leave displayed adapters/previews on the previous registry even though
the manager and saved state have changed. Schema3 makes that stale capability
visible and relevant to the new lifecycle contract.

Astra owns the narrow correction and its one meaningful regression as part of
current app implementation. Distinguish persistence refusal (live state/views
and drafts kept) from successfully persisted settings followed by load failure
(new registry and its usable entries must be reflected, error retained). Use
an explicit manager outcome/changed flag if needed; do not infer all errors
mean rollback and do not unconditionally release on persistence errors. Existing
SetEnabled API can remain as compatibility wrapper if the app needs a new
additive action outcome. Avoid a generic transaction framework.

Use a real installed file that is deleted/corrupted before enabling and an
already-active route, assert manager/adapter state agrees and late worker results
cannot attach. Existing settings-file persistence-failure tests must still show
preserved routing/drafts. Report exact scope if core source changes after its
accepted gate; rerun only affected core tests plus app tests/vet, not all modules.
