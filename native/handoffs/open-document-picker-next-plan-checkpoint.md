# Open-document picker planning checkpoint

2026-11-10 Astra; read-only source inspection only. Paused for root-activated
composition final validation. No feature implementation/tests/product runs.

Actual seams inspected: App.Tabs has maximum100live records and stable int Tab.ID;
title Tab.Name, text/binary-neutral Tab.path() in app/binary.go. Current()/Select/
selectChecked/flushTab in app.go own real switch/release/persistence. Picker must
retain IDs+bounded metadata only, resolve index at activation and revalidate ID
after Flush via existing selectChecked callback; indices are not identities.
Select can synchronously flush/reenter, so plain pre-resolved index is insufficient.

Proposed entry: compact “已打开文档” button next to tab strip, Ctrl+P currently
unused in app shortcuts. Native in-content panel, title/path substring filtering,
stable tab order, no files/disk/index. One text input and fixed-height virtual
List; MyGo ListState.Key/Label/Selected and Element.Submitted are public; list
builds visible rows only. Distinguish same names with bounded path+stable tab ID.

Composition patterns: sourceCompositionActive() checks source MarkedRange/rich
Composer, but first rejected preedit has no mark: opening also needs editor
Flush error gating after active-compose check. Utility Find/GoToLine already
observe InputCompose/InputText and reconcile actual Focused/Composing on pointer
leave; reuse policy. No opening/focus move while source/rich/utility composition
active; no queue. Query preedit must block activation and panel closure while
native candidate input is unresolved. Source Esc stays with its local editor.

Still to finalize after validation: exact query/list Up/Down public reveal API,
pointer KeepFocus row route, bounded Unicode matching/detail limits and focus
restoration policy for preview and utility origins (no foreign editor pointers).
Task target remains open-document-picker-next-plan.md; .29 closure is required
before any implementation.
