#!/usr/bin/env bash
# Repository-local Linux development entrypoint (Go 1.27.3, patched vendor).
set -euo pipefail
levnote_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
source "$levnote_root/.localtools/env.sh"
levnote_go="$GOROOT/bin/go"
levnote_version="$("$levnote_go" version)"
case "$levnote_version" in
  'go version go1.27.3 '*) ;;
  *) printf 'Expected Go 1.27.3; got %s\n' "$levnote_version" >&2; exit 2 ;;
esac
levnote_command="${1:-help}"
if (( $# )); then shift; fi
case "$levnote_command" in
  version|env|help) ;;
  build|test|vet|run|list) ;;
  *) printf 'Unsupported command: %s\n' "$levnote_command" >&2; exit 2 ;;
esac
for levnote_arg in "$@"; do
  case "$levnote_arg" in
    -C|--C|-C=*|--C=*|-modfile*|--modfile*|-overlay*|--overlay*|-mod|--mod)
      printf 'Alternate build inputs are unsupported: %s\n' "$levnote_arg" >&2; exit 2 ;;
    -mod=*|--mod=*)
      if [[ "$levnote_arg" != '-mod=vendor' && "$levnote_arg" != '--mod=vendor' ]]; then
        printf 'Vendor mode is required: %s\n' "$levnote_arg" >&2; exit 2
      fi ;;
    ./...)
      printf 'Use the six product roots; ./... includes historical handoff snapshots.\n' >&2; exit 2 ;;
  esac
done
if [[ "$levnote_command" != version && "$levnote_command" != env && "$levnote_command" != help ]]; then
  node --input-type=module - "$levnote_root/native" <<'JS'
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync, lstatSync } from 'node:fs';
import path from 'node:path';
const root = process.argv[2];
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const bytes = readFileSync(path.join(root, 'dependency-patches/mygo-v0.8.3-patch-files.json'));
assert.equal(hash(bytes), '9113120bd2ebcd9a86842153239cddff84d1f1bd3f46f5d4f0f152eb4560e16a', 'MyGo patch manifest pin changed');
const manifest = JSON.parse(bytes);
assert.equal(manifest.schema, 1);
assert.equal(manifest.module, 'github.com/egoist/mygo v0.8.3');
assert.equal(manifest.files.length, 26);
for (const entry of manifest.files) {
  assert.equal(hash(readFileSync(path.join(root, entry.path))), entry.sha256, `MyGo patch input changed: ${entry.path}`);
}
const nativeManifestPath = path.join(root, 'dependency-patches/mygo-v0.8.3-native-input-files.json');
assert.ok(lstatSync(nativeManifestPath).isFile() && !lstatSync(nativeManifestPath).isSymbolicLink(), 'Native-input supplement must be a regular file');
const nativeBytes = readFileSync(nativeManifestPath);
assert.equal(hash(nativeBytes), '12553bf8918bf9a56a2c8384e324633145eceb41685a0c982666d59b80327fe5', 'Native-input supplement pin changed');
const supplement = JSON.parse(nativeBytes);
assert.equal(supplement.schema, 1);
assert.equal(supplement.module, 'github.com/egoist/mygo v0.8.3');
const expectedNative = [
  'dependency-patches/mygo-v0.8.3-native-input.patch',
  'vendor/github.com/egoist/mygo/internal/platform/text_input.go',
  'vendor/github.com/egoist/mygo/internal/windows/surface.go',
  'vendor/github.com/egoist/mygo/internal/windows/surface_input.go',
  'vendor/github.com/egoist/mygo/ui/headless.go',
  'vendor/github.com/egoist/mygo/ui/text_input_cancel_test.go',
  'vendor/github.com/egoist/mygo/ui/text_input_client.go',
].sort();
assert.deepEqual(supplement.files.map(entry => entry.path).sort(), expectedNative, 'Native-input supplement membership must be exact');
const oldPaths = new Set(manifest.files.map(entry => entry.path.toLowerCase()));
for (const entry of supplement.files) {
  assert.ok(typeof entry.path === 'string' && !entry.path.includes('\\') && !entry.path.split('/').some(part => !part || part === '.' || part === '..'));
  assert.ok(!oldPaths.has(entry.path.toLowerCase()), 'Old/native supplement overlap');
  const member = path.join(root, entry.path);
  assert.ok(lstatSync(member).isFile() && !lstatSync(member).isSymbolicLink(), 'Native-input member must be a regular file');
  assert.match(entry.sha256, /^[0-9a-f]{64}$/);
  assert.equal(hash(readFileSync(member)), entry.sha256, `Native-input changed: ${entry.path}`);
}
const app = readFileSync(path.join(root, 'vendor/github.com/egoist/mygo/app.go'), 'utf8');
assert.ok(app.includes('// LEVNOTE PATCH: this app does not enable MyGo updates'));
assert.ok(!/\bgo\s+cleanUpdateLeftovers\s*\(/.test(app));
assert.ok(readFileSync(path.join(root, 'vendor/modules.txt'), 'utf8').includes('# github.com/egoist/mygo v0.8.3'));
JS
fi
levnote_roots=(./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...)
if [[ "$levnote_command" == build || "$levnote_command" == vet || "$levnote_command" == test ]] && (( $# == 0 )); then
  set -- "${levnote_roots[@]}"
fi
cd -- "$levnote_root/native"
exec "$levnote_go" "$levnote_command" "$@"
