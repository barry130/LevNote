# Paint-independent recovery and image keyboard source accepted

2026-11-10 root reviewed current runtime, substantive tests, exact commands,
preserved failures and final pre/post15file inventories. Accepted for .24 only;
the existing .23 artifact is unchanged.

Automatic recovery uses one cancellable timer, captured Window.Update dispatcher
and UI-owned state checks; the timer thread only posts. No View is required.
Edits retain900ms debounce; same deadline coalesces; late/early callbacks, nested
modal exit, Restore, explicit persistence and Shutdown follow generation/lifecycle
rules. Failures make one automatic attempt until mutation or explicit retry.
No automatic Flush or composition commit, no write to original document files;
format-local uncommitted drafts remain outside this recovery contract. Headless
inline uiDispatch never arms the real timer. Binary recovery remains metadata.

Focused image canvas handles+/=/-/keypad,0fit,1manual100% and32DIP arrows with
exact modifiers, key/code single action and drag isolation. Toolbar actions
return canvas focus. Text/IME, bytes, other tabs and released views retain their
own routes. Relative clamped no-ops preserve fit. Root found explicit100% at
fit1 left fitMode true; narrow correction and meaningful400x200 resize regression
now distinguish cropped manual100% from public0fit. The first combined failure
and the initially weaker corrected regression remain recorded.

Final affected command through scripts/go.ps1:
test ./app ./modules/imageview -count=1 -> exit0, app23.863s,imageview15.447s.
vet ./app ./modules/imageview -> exit0,empty output. No repeated passing suite.
Evidence native-autosave-evidence-20261110/08-corrected-final-test.* and
09-final-vet.*;07/10pre/post15source entries all match. Earlier eight autosave
tests passed first focused run. The original04combined failure is preserved.

Final validation handoff SHA256
D747A8CD70596D6D9C6E3B58C359E0408A6C9ED707EA260F3D9D810E74DF2281.
Autosave11source identities are in source-freeze.json; final image source:
input.go2963B 76D1AB17A3F9B4DC99A153782305AB490904F547A6C7239F2A548248BB2601DE;
ui.go6456B BEF3A43EA6A028791A623728E9899CF481A1BB46BC12D91D61AFD67FB95AA368;
imageview_test.go42807B 824645F0115D8E4DF7851696E78B33AAC941523C9C2D5929B0EB3B1019094231;
binary_keyboard_test.go5793B 77856CCE4BA1FBA6E2FF2E1E6044C0149FC145948320F8E5171BFA3BF137050D.

Synthetic Tester and fake-time/UI-queue evidence do not establish physical
Windows minimize/IME, GPU, RSS/VRAM or CPU behavior. Root ran no checks/builds.
Runtime stays frozen through .24 delivery. native-reading-release24-task.md is
now ACTIVE for its bounded package/one-host execution after complete helpers
and portable README are prepared, parsed and internally reviewed by its owner.
No further root permission is required for the operations specified there.
