# Binary session implementation review while work is active

2026-11-10 root read-only review. Fix these concrete issues before final checks;
no expanded feature scope or unrelated benchmark suite is needed.

1. output.go currently wraps every mixed-session entry in tabWire.MarshalJSON,
   which calls json.Marshal(t.Text). This is the same extra full draft encoding
   and returned byte copy forbidden by the task, just on a wrapper instead of
   Tab itself. Use bounded []any projection containing ordinary Tab values and
   small binaryTabWire values directly; the stdlib encoder handles each concrete
   value without a per-entry custom MarshalJSON. Text-only path stays unchanged.
   Do not retain a custom marshal wrapper around ordinary text entries.
2. load.go objectKeys uses Decoder.Decode into json.RawMessage for every value.
   This copies an entire draft (and decoder buffering can copy it again) before
   the real text unmarshal, contradicting the task's one draft-sized decode.
   Scan the already validated per-entry JSON bytes by offsets to capture key
   names and zero-copy value slices. Skip quoted/nested values correctly, decode
   only small key/discriminator strings, then unmarshal the ordinary Tab once.
   Do not allocate RawMessage copies of draft or a map of copied values. The
   existing boundedTabs scanner is a useful model. Preserve proper escaped-key
   comparison and duplicate-field decisions.

Retain substantive text/mixed wire oracles and actual preflight length tests.
Add a narrow structural assertion or a clear code-based verification of the
absence of custom text marshaling and draft RawMessage copying; do not replace
the behavior tests with structural checks or invent performance measurements.

Follow-up after scanner draft appeared: current value loop decrements depth on
the OUTER closing brace, reaching -1 instead of stopping; it also stops at the
closing brace of a NESTED object when depth becomes0, excluding that object's
closing byte. Check a delimiter at depth0 before consuming it (as boundedTabs
does); nested closing braces are part of the value. Last primitive/string and
nested selection/array/unknown fields need meaningful tests. Also remove the
production sessionWireOracle alias: an alias to the implementation wire is not
an independent test oracle and adds an unnecessary product declaration. Declare
oracle structs separately in the test file.

Current11:34 test review (source only, no root checks):
- binary_structural_test.go currently marks hasRawMessage=true merely on importing
  encoding/json, so it unconditionally fails this valid package. Inspect the
  selector RawMessage itself, not the import, or use a simpler exact check.
- TestBinaryRecordStrictWireContract unknown-nested-array fixture has an object
  key/value inside an array without its object braces, plus mismatched closing
  braces. Construct a valid nested object/array from a separately declared map
  or struct with json.Marshal instead of repeatedly editing JSON punctuation.
- Mixed oracle still uses production binaryTabWire; declare its small binary
  entry independently in test so a tag/omission regression cannot change both
  the implementation and expected bytes together.
- Add valid escaped key/discriminator and ordinary text nested selection,
  last scalar/string/array cases to the scanner regressions, including proof
  draft value slices alias existing input rather than copying.
