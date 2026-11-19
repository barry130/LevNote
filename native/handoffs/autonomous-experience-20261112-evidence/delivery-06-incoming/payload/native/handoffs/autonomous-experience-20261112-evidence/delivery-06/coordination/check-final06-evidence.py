#!/usr/bin/env python3
"""Audit saved final06 test inputs and outcomes; no product execution or source writes."""
from pathlib import Path
import hashlib
import json
import sys

LABELS = {'experience-list-final06-focused-01', 'experience-list-final06-affected-01'}
assert len(sys.argv) == 2 and sys.argv[1] in LABELS
label = sys.argv[1]
BASE = Path('/home/user/levnote-dev-reliability-20261112-01')
ROOT = Path('/home/user/levnote-dev-experience-20261112-01')
ev = BASE / 'runs' / label / 'evidence'
source = BASE / 'runs' / label / 'source/native'
expected_path = BASE / 'inputs' / label / 'expected-manifest.json'
freeze_path = ROOT / 'provenance/task6-freeze-06/source-inputs.json'
review_path = ROOT / 'coordination/resume-06-input-review-20261113-01.json'


def ident(path):
    data = path.read_bytes()
    return {'path': str(path), 'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}


def parsed(path):
    return json.loads(path.read_bytes())


def small_id(path):
    result = ident(path)
    return {key: result[key] for key in ['bytes', 'sha256']}


assert small_id(freeze_path) == {'bytes': 4605, 'sha256': 'f864c426b7d200533641839fea12b4907241050778ba46a4470256773fbbb2dd'}
freeze, review, expected = parsed(freeze_path), parsed(review_path), parsed(expected_path)
assert review['freeze06'] == ident(freeze_path)
assert expected['head'] == freeze['baseline'] == '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
wr, num = parsed(ev / 'result.json'), parsed(ev / '05-product-command.json')
start = parsed(ev / '05-product-command.started.json')
intent = parsed(ev / '05-product-command.command.json')
assert wr['product_command'] == num
assert num['status'] == 'completed'
assert num['pid'] == num['owned_pgid'] == start['pid'] == start['owned_pgid']
assert num['argv'] == start['argv'] == intent['argv']
assert num['cwd'] == start['cwd'] == str(source)
assert num['timed_out'] is False and num['owned_cleanup'] == [] and num['owned_group_remaining_after_wait'] is False
assert (num['exit_code'], wr['exit_code'], wr['outcome']) in [(0, 0, 'COMMAND_PASSED_LIMITED_SCOPE'), (1, 2, 'COMMAND_FAILED')]
assert wr['runner_input'] == ident(Path(wr['runner_input']['path']))
assert wr['runner_input']['bytes'] == 17964 and wr['runner_input']['sha256'] == '6f3dd79076e9f284691f24be1a8e3b0c91d57d9c0ed5766d07cc87445acdff89'
assert wr['ordinary_env_input'] == ident(Path('/home/user/levnote-dev-env.sh'))
assert wr['ordinary_env_input']['bytes'] == 175 and wr['ordinary_env_input']['sha256'] == '9b6b8b1aad396bd0add1d2f469de218ae3df99079b72da1b21e79f99f0b51e47'
assert wr['effective_go_environment']['GOVERSION'] == 'go1.27.3'
for key, value in {'GOFLAGS': '-mod=vendor', 'GOPROXY': 'off', 'GOSUMDB': 'off', 'GOTOOLCHAIN': 'local', 'GOWORK': 'off', 'GOVCS': '*:off', 'CGO_ENABLED': '0', 'GOOS': 'linux', 'GOARCH': 'amd64', 'GOTELEMETRY': 'off'}.items():
    assert num['environment'][key] == value
assert wr['expected_input'] == ident(expected_path)
assert wr['not_windows_or_whole_application_acceptance'] is True
map_names = ['source-before.json', 'source-after.json', 'snapshot-after.json']
map_ids = {name: ident(ev / name) for name in map_names}
assert len({item['sha256'] for item in map_ids.values()}) == 1
assert wr['source_before_manifest'] == small_id(ev / 'source-before.json')
assert len(expected['files']) == 1194 and sum(item['bytes'] for item in expected['files'].values()) == 17000651
for name in map_names:
    saved = parsed(ev / name)
    assert saved['roots'] == expected['roots'] and saved['files'] == expected['files']
# Audit every input in both caller input and finished isolated product snapshot.
for rel, want in expected['files'].items():
    assert small_id(source / rel) == want
    assert small_id(BASE / 'inputs' / label / 'native' / rel) == want
    assert small_id(ROOT / 'revision-06-source/native' / rel) == want
nine = []
for f in freeze['files']:
    rel = f['path'].removeprefix('native/')
    want = {key: f[key] for key in ['bytes', 'sha256']}
    assert expected['files'][rel] == want
    nine.append({'relative': rel, **ident(source / rel)})
assert expected['files']['app/list_handle_lifetime_test.go'] == {'bytes': 7900, 'sha256': 'ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5'}
for stream in ['stdout', 'stderr']:
    assert num[stream] == ident(Path(num[stream]['path']))
argv = num['argv']
assert argv[:4] == ['/home/user/go/bin/go', 'test', '-v', '-count=1'] and argv[-1] == './app'
selector = argv[argv.index('-run') + 1]
focused = label.endswith('focused-01')
if focused:
    assert argv[4] == '-timeout=120s' and num['timeout_s'] == 180
    assert selector == '^TestWorkspaceResultsMinimumWindowControls$'
    wanted = ['TestWorkspaceResultsMinimumWindowControls']
else:
    assert argv[4] == '-timeout=180s' and num['timeout_s'] == 240
    assert selector == review['original25_selector']
    wanted = selector.removeprefix('^(').removesuffix(')$').split('|')
    assert len(wanted) == len(set(wanted)) == 25
raw = Path(num['stdout']['path']).read_text()
lines = raw.splitlines()
runs = [line.removeprefix('=== RUN   ') for line in lines if line.startswith('=== RUN   ') and '/' not in line]
passes = [line.split(': ', 1)[1].rsplit(' (', 1)[0] for line in lines if line.startswith('--- PASS: ')]
fails = [line.split(': ', 1)[1].rsplit(' (', 1)[0] for line in lines if line.startswith('--- FAIL: ')]
nested_runs = [line.removeprefix('=== RUN   ') for line in lines if line.startswith('=== RUN   ') and '/' in line]
nested_pass = [line.strip().split(': ', 1)[1].rsplit(' (', 1)[0] for line in lines if line.startswith('    --- PASS: ')]
nested_fail = [line.strip().split(': ', 1)[1].rsplit(' (', 1)[0] for line in lines if line.startswith('    --- FAIL: ')]
panic = any(line.startswith('panic:') for line in lines)
compile_failure = any('[build failed]' in line for line in lines)
if num['exit_code'] == 0:
    assert not panic and not compile_failure and not fails and not nested_fail
    assert len(runs) == len(passes) == len(wanted) and set(runs) == set(passes) == set(wanted)
    assert 'PASS' in lines
    if focused:
        assert len(nested_runs) == len(nested_pass) == 2
        assert set(nested_pass) == {'TestWorkspaceResultsMinimumWindowControls/dark=false', 'TestWorkspaceResultsMinimumWindowControls/dark=true'}
    else:
        assert len(nested_runs) == len(nested_pass) == 22
renders = [line.strip() for line in lines if 'public Tester development render:' in line]
print(json.dumps({
    'status': 'SAVED06_METADATA_AUDITED_OUTCOME_NOT_APP_ACCEPTANCE', 'label': label,
    'pid': num['pid'], 'product_exit': num['exit_code'], 'wrapper_exit': wr['exit_code'], 'outcome': wr['outcome'],
    'launched_at': num['launched_at'], 'ended_at': num['ended_at'], 'elapsed_s': num['elapsed_s'],
    'timed_out': num['timed_out'], 'cleanup': num['owned_cleanup'], 'remaining_group': num['owned_group_remaining_after_wait'],
    'identities': {name: ident(ev / name) for name in map_names + ['result.json', '05-product-command.json', '05-product-command.started.json', '05-product-command.command.json', '05-product-command.stdout.bin', '05-product-command.stderr.bin']},
    'expected': ident(expected_path), 'same_three_maps_roots_and_every_file': True,
    'all1194_actual_input_and_snapshot_files_exact_unformatted06': True,
    'mapped_files': len(expected['files']), 'mapped_bytes': sum(x['bytes'] for x in expected['files'].values()),
    'nine_actual_inputs_equal_freeze': nine, 'oracle7900_unchanged': True,
    'top_level_started': runs, 'top_level_passed': passes, 'top_level_failed': fails,
    'nested_started': len(nested_runs), 'nested_passed': len(nested_pass), 'nested_failed': len(nested_fail),
    'panic': panic, 'compile_failure': compile_failure, 'never_started': sorted(set(wanted) - set(runs)),
    'public_render_log_lines': renders, 'render_files_retained_and_visual_reviewed': False,
    'author_product_commands': 0, 'not_windows_or_whole_application_acceptance': True,
}, ensure_ascii=False, indent=2))
