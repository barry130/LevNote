# Fourth worker stage 2 progress

ACTIVE 2026-11-10. Implementation source now compiles for
`./modules/graphview` and `./app`; this milestone did not execute tests.

Implemented so far:

- graphview draw.io advertises the fourth fixed `levnote.drawio-svg` engine with
  `RequiresView=true`;
- app export captures mode/view/page/token, prepares after Flush and source
  limits, revalidates all identities, and carries an independent
  `engineipc.RequestPage` into the worker request;
- result and Save-dialog rechecks include the side-effect-free draw.io epoch;
- draw.io source mode and a missing prepared view are visibly refused;
- the package panel exposes all four fixed engines with count-neutral reset
  wording.

Compile command:

```text
.\native\scripts\go.ps1 -GoArguments @('test','./modules/graphview','./app','-run','^$','-count=1')
```

Exit `0`; output:

```text
ok  levnote.local/native/modules/graphview  1.281s [no tests to run]
ok  levnote.local/native/app               1.592s [no tests to run]
```

Still pending: decisive stage-2 tests, one final affected suite/vet, and two
760x480 captures. No `.19` build/package/launch performed.
