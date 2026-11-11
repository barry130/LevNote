# Native app store integration tests — DeepSeek

2026-11-09. Astra failed model capacity. New GLM owns all app production and
lifecycle tests. Your exclusive scope is new native/app/engine_packages_store_
test.go + handoffs/engine-app-store-tests.md. No other Go edits or child agents.
Read current engine_packages.go/UI and existing app test helpers; implement now.
Do not repeat your withdrawn review findings. Compact before200K.

Author a small coherent real filesystem release fixture for all needed packages
(inert exe text NEVER executed), based on exact18file store contract and current
catalog/manifest/inventory. Can use public PackageManager to install/activate;
do not export private verifier bypasses. Helper `enginePackageFixture(t, engine,
version) string` may be shared with GLM; publish that exact helper name promptly.
The real store itself is already tested; don't replay its decoder/cap/link matrix.

Test actual App.startPackageWork/deliverPackageWork with done bounded waits:
import does not activate; cached receipt accessible; explicit activation/reset
changes only intended engine; new App loads persisted choice; list returns
healthy packages plus visible corruption error; corrupt state stays blocked and
explicit ALL3 repair retains imported package; failure leaves selection/status
truthful.3-4 cohesive tests suffice. No fake successful Manager, no build/real
engine/OS UI/resource work. Current app init uses os.Executable and temp config.

Do NOT run go tests while GLM source may be mid-edit. First complete/gofmt this
file and publish frozen helper/API/results-not-yet-run handoff. Notify root/GLM
through collaboration only (no app thread messaging). GLM will run combined
focused/full app once all files compile. If you find actual production bug,
report exact source/trigger to root; never patch peer product files. Final text
must distinguish written tests from actual passed evidence. Root reads source.
