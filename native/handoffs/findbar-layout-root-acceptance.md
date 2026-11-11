# Find-bar reflow: root acceptance

2026-11-09. Root accepts the minimal Find-bar change for .13 delivery.
Current runnable binary remains .12 until the candidate is separately accepted.

Root reviewed actual state.go/test source, original failing geometry, final
focused/app/vet logs, owner handoff and both actual760x480 PNGs. Root recomputed
the two source/test and two image hashes; they match findbar-layout-result.md.
Accepted search_navigation.go remains unchanged with SHA256
8F77367B137135618CEB65BE0513ED5670033E2500A867C87DD8185CB5854E6D.

Only two production Wrap calls changed. Original regex control ended at x809
outside760; all controls now end by x748, last action row y238.041. Editor
area532x203.959 at(228,246.041), above fixed status top451. Both themes and
maximum32px editor-font geometry covered; screenshots visibly legible. Resize
preserves query/replacement, focus and toolkit composition; navigation/options/
replace/close actions and Tab traversal passed. No new keyboard semantics.

Final focused2.213s; affected full app16.235s; scoped vet exit0. Harness-role
mismatch and capture-path correction are separately documented; original
overflow red retained. No build/physical OSIME/resource claim from these tests.
Root did not run implementation/tests/builds. Source/test hashes:

- state.go59E1DF87541AC786D2CE5603AD10CBCE14CF4B1D9775358B1241E66FAB461D7E
- findbar_layout_test.goF20B0A9F777D8DE80632092DB3A1BE5FAEEF5E32AF392FF9DF03896C82193D1C

Activate search-navigation-release13-delivery-task.md. All product source/tests
and accepted screenshots are frozen during delivery. Separate keyboard review
is a future stage, not .13 scope.
