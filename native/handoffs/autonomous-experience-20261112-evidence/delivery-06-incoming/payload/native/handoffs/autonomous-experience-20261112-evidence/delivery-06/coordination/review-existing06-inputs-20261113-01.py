#!/usr/bin/env python3
"""Read-only revalidation of existing06; writes only a new metadata receipt, no Go."""
from pathlib import Path
import datetime
import difflib
import hashlib
import json

ROOT = Path('/home/user/levnote-dev-experience-20261112-01')
OLDROOT = ROOT / 'revision-05-source'
NEWROOT = ROOT / 'revision-06-source'
ORACLE_ROOT = ROOT / 'revision-03-source'
OLD_FREEZE = ROOT / 'provenance/task6-freeze-05/source-inputs.json'
NEW_FREEZE = ROOT / 'provenance/task6-freeze-06/source-inputs.json'
PATCH = ROOT / 'provenance/task6-freeze-06/revision-05-to-06.patch'
READY = ROOT / 'coordination/final06-ready.txt'
RED = Path('/home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final05-affected-01')
OUTPUT = ROOT / 'coordination/resume-06-input-review-20261113-01.json'
REL = 'native/app/workspace.go'
OLD = b'\tui.Column(c).Height(panelHeight).Shrink(0).Border(1, c.Theme().Border).Children(func() {\n'
NEW = b'\tui.Column(c).Height(panelHeight).Shrink(1).MinHeight(110).Border(1, c.Theme().Border).Children(func() {\n'


def pair(data):
    return {'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}


def ident(path):
    return {'path': str(path), **pair(path.read_bytes())}


def read_json(path):
    return json.loads(path.read_bytes())


def inventory(root):
    result = {}
    for path in root.rglob('*'):
        assert not path.is_symlink(), str(path)
        if path.is_file():
            result[path.relative_to(root).as_posix()] = pair(path.read_bytes())
    return result


assert not OUTPUT.exists(), 'new review receipt must be create-only'
fixed_inputs_before = {path: ident(path) for path in [OLD_FREEZE, NEW_FREEZE, PATCH, READY]}
assert pair(OLD_FREEZE.read_bytes()) == {'bytes': 3036, 'sha256': '74ec096142fd683f4deeea41c59c66e2697accb068ed8c48c448e1c1065a74e7'}
assert pair(NEW_FREEZE.read_bytes()) == {'bytes': 4605, 'sha256': 'f864c426b7d200533641839fea12b4907241050778ba46a4470256773fbbb2dd'}
assert pair(PATCH.read_bytes()) == {'bytes': 517, 'sha256': 'a24f2833b397deccf45c308c6ddd3b743fba4fdb253419fd86a19b130405a64e'}
previous, freeze = read_json(OLD_FREEZE), read_json(NEW_FREEZE)
assert freeze['parent_freeze'] == ident(OLD_FREEZE)
assert freeze['parent_source_root'] == str(OLDROOT) and freeze['source_root'] == str(NEWROOT)
assert freeze['baseline'] == previous['baseline'] == '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
assert freeze['status'] == 'FROZEN_UNFORMATTED_UNEXECUTED_NOT_ACCEPTANCE'
before, after = inventory(OLDROOT), inventory(NEWROOT)
assert before.keys() == after.keys()
assert len(before) == previous['source_files'] == len(after) == freeze['source_files'] == 5967
assert sum(x['bytes'] for x in before.values()) == previous['source_bytes'] == 80238552
assert sum(x['bytes'] for x in after.values()) == freeze['source_bytes'] == 80238567
changed = sorted(rel for rel in before if before[rel] != after[rel])
assert changed == freeze['changed_from05'] == [REL]
old, new = (OLDROOT / REL).read_bytes(), (NEWROOT / REL).read_bytes()
assert old.count(OLD) == 1 and new == old.replace(OLD, NEW, 1)
assert PATCH.read_bytes() == ''.join(difflib.unified_diff(old.decode().splitlines(True), new.decode().splitlines(True), fromfile='a/' + REL, tofile='b/' + REL)).encode()
assert [f['path'] for f in previous['files']] == [f['path'] for f in freeze['files']]
for f in previous['files']:
    assert before[f['path']] == {k: f[k] for k in ['bytes', 'sha256']}
for f in freeze['files']:
    rel = f['path']
    assert after[rel] == {k: f[k] for k in ['bytes', 'sha256']}
    assert f['identical_to05'] == (rel != REL)
tests = sorted(rel for rel in after if rel.endswith('_test.go'))
for rel in tests:
    assert after[rel] == before[rel] == pair((ORACLE_ROOT / rel).read_bytes()), rel
assert after['native/app/list_handle_lifetime_test.go'] == {'bytes': 7900, 'sha256': 'ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5'}
source_map = read_json(RED / 'evidence/source-before.json')
expected_path = Path('/home/user/levnote-dev-reliability-20261112-01/inputs/experience-list-final05-affected-01/expected-manifest.json')
expected = read_json(expected_path)
for name in ['source-before.json', 'source-after.json', 'snapshot-after.json']:
    saved = read_json(RED / 'evidence' / name)
    assert saved['roots'] == expected['roots'] and saved['files'] == expected['files']
assert len(source_map['files']) == 1194 and sum(x['bytes'] for x in source_map['files'].values()) == 17000636
for rel, p in source_map['files'].items():
    assert before['native/' + rel] == p
    assert pair((RED / 'source/native' / rel).read_bytes()) == p
    assert after['native/' + rel] == (after[REL] if rel == 'app/workspace.go' else p)
rednum = read_json(RED / 'evidence/05-product-command.json')
assert rednum['pid'] == rednum['owned_pgid'] == 31061 and rednum['exit_code'] == 1
selector = rednum['argv'][rednum['argv'].index('-run') + 1]
ready_lines = READY.read_text().splitlines()
selectors = [line for line in ready_lines if line.startswith('^(') and line.endswith(')$')]
assert selectors == [selector]
assert 'go test -v -count=1 -timeout=120s -run ^TestWorkspaceResultsMinimumWindowControls$ ./app' in ready_lines
assert len(selector.removeprefix('^(').removesuffix(')$').split('|')) == 25
for f in freeze['files']:
    assert '{} {} {}'.format(f['path'], f['bytes'], f['sha256']) in ready_lines
# Recheck the fixed inputs after reading the entire ancestry; no source edit is performed.
for path, want in fixed_inputs_before.items():
    assert ident(path) == want
receipt = {
    'status': 'EXISTING06_REVALIDATED_METADATA_ONLY_NOT_PRODUCT_ACCEPTANCE',
    'reviewed_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'reviewer': '9e20bc49-fddb-490b-a752-c3b11da86ccc/task6',
    'helper': ident(Path(__file__)),
    'parent_root': str(OLDROOT), 'source_root': str(NEWROOT),
    'freeze05': ident(OLD_FREEZE), 'freeze06': ident(NEW_FREEZE), 'delta': ident(PATCH), 'ready': ident(READY),
    'whole_source_files': len(after), 'whole_source_bytes05': sum(x['bytes'] for x in before.values()),
    'whole_source_bytes06': sum(x['bytes'] for x in after.values()),
    'exact05_file_set_and_every_nonchanged_byte': True, 'only_changed_file': REL,
    'exact_one_line_replacement': True, 'eight_other_scoped_inputs_exact05': True,
    'all_test_files_exact03_and05': True, 'test_files_checked': len(tests), 'oracle7900_unchanged': True,
    'actual31061_source_before_roots_and_every_mapped_file_exact05': True,
    'red_mapped_files': len(source_map['files']), 'red_mapped_bytes': sum(x['bytes'] for x in source_map['files'].values()),
    'same_actual_red_source_map_except_one_workspace_line': True,
    'original25_selector_byte_exact31061': True, 'original25_selector': selector,
    'nine_inputs': freeze['files'],
    'author_product_commands': 0, 'freeze_reexecution': False,
    'current_run_status_source': 'Soleowner status-first reports no06 input/intent/started/final or product PID; new validation not launched by this reviewer.',
    'not_windows_or_whole_application_acceptance': True,
}
with OUTPUT.open('xb') as out:
    out.write((json.dumps(receipt, ensure_ascii=False, indent=2) + '\n').encode())
print(json.dumps({'receipt': ident(OUTPUT), **receipt}, ensure_ascii=False, indent=2))
