#!/usr/bin/env python3
"""Create a fresh pinned input from an owned, successful gofmt snapshot; no Go."""
import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import traceback

LANE = Path('/home/user/levnote-dev-reliability-20261112-01')
HELPER = LANE / 'tools/dev-remote-validation-20261112-03.py'
HEAD = '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--format-label', required=True)
p.add_argument('--label', required=True)
a = p.parse_args()
if any(not re.fullmatch(r'[a-z0-9][a-z0-9-]{0,79}', label) for label in (a.format_label, a.label)) or a.format_label == a.label:
    p.error('two distinct safe labels are required')
os.umask(0o077)
spec = importlib.util.spec_from_file_location('runner02', HELPER)
r = importlib.util.module_from_spec(spec)
spec.loader.exec_module(r)
run = LANE / 'runs' / a.format_label
original = LANE / 'inputs' / a.format_label
formatted = run / 'source/native'
evidence = LANE / 'evidence' / ('derive-' + a.label)
evidence.mkdir(parents=True, exist_ok=False)
rec = {'argv': sys.argv, 'pid': os.getpid(), 'cwd': str(Path.cwd()), 'started_at': r.utc(), 'script': r.file_identity(__file__), 'format_label': a.format_label, 'product_executed': False}
r.write_json(evidence / 'started.json', rec)
code = 2
try:
    before = json.loads((original / 'expected-manifest.json').read_bytes())
    command = json.loads((run / 'evidence/05-product-command.json').read_bytes())
    if before['head'] != HEAD or command['exit_code'] != 0 or command['timed_out'] or command['owned_group_remaining_after_wait'] or command['owned_cleanup'] or command.get('owned_cleanup_error') or command.get('started_receipt_error'):
        raise ValueError('owned format command did not finish cleanly at the pinned baseline')
    if command['argv'][0] != '/home/user/go/bin/gofmt' or command['argv'][1] != '-w':
        raise ValueError('source is not the recorded owned gofmt invocation')
    allowed = set(command['argv'][2:])
    if any((not name.startswith('app/') and name not in ('formats/drawio/delete.go', 'formats/drawio/delete_test.go', 'modules/graphview/drawio_delete_test.go', 'modules/graphview/input.go', 'modules/graphview/drawio_delete.go', 'core/document/document.go', 'core/document/atomic.go', 'core/document/utf8_save_test.go', 'core/document/fingerprint_boundary_test.go', 'core/document/fingerprint_public_test.go')) or not name.endswith('.go') for name in allowed):
        raise ValueError('only assigned app/drawio or the four exact document-boundary files may be derived')
    raw = r.inventory(original / 'native')
    after = r.inventory(formatted)
    if raw['files'] != before['files'] or set(raw['files']) != set(after['files']):
        raise ValueError('raw input changed or gofmt added/deleted files')
    changes = {name for name in raw['files'] if raw['files'][name] != after['files'][name]}
    if not changes.issubset(allowed):
        raise ValueError('format snapshot changed outside the explicit formatter group')
    native = LANE / 'inputs' / a.label / 'native'
    native.mkdir(parents=True, exist_ok=False)
    for name in after['files']:
        target = native / name
        target.parent.mkdir(parents=True, exist_ok=True)
        with target.open('xb') as f:
            f.write((formatted / name).read_bytes())
    if r.inventory(formatted) != after or r.inventory(native) != after or r.inventory(original / 'native') != raw:
        raise ValueError('raw, format snapshot or copied input changed during capture')
    overlay = [dict(row, input=after['files'][row['path']]) for row in before['overlay']]
    expected = {'head': HEAD, **after, 'source_origin': str(formatted), 'owner': 'native-reliability/sole-product-lane', 'label': a.label, 'overlay': overlay, 'input_count': len(after['files']), 'input_bytes': sum(v['bytes'] for v in after['files'].values()), 'format_evidence': str(run / 'evidence')}
    manifest = native.parent / 'expected-manifest.json'
    r.write_json(manifest, expected)
    r.write_json(evidence / 'raw-before.json', raw)
    r.write_json(evidence / 'formatted-before.json', after)
    r.write_json(evidence / 'raw-after.json', r.inventory(original / 'native'))
    r.write_json(evidence / 'formatted-after.json', r.inventory(formatted))
    rec.update({'outcome': 'MATCHING_FORMATTED_INPUT_CAPTURED_NOT_PRODUCT_ACCEPTANCE', 'input_source': str(native), 'expected_manifest': {'path': str(manifest), **r.file_identity(manifest)}, 'input_count': expected['input_count'], 'input_bytes': expected['input_bytes'], 'formatter_changed_files': sorted(changes), 'formatted_file_group': {name: after['files'][name] for name in sorted(allowed)}, 'raw_manifest': r.file_identity(original / 'expected-manifest.json')})
    code = 0
except BaseException:
    error = traceback.format_exc()
    with (evidence / 'wrapper.stderr.txt').open('x', encoding='utf-8') as f:
        f.write(error)
    print(error, flush=True)
    rec['outcome'] = 'FORMATTED_CAPTURE_FAILED_NO_GO'
rec.update({'exit_code': code, 'ended_at': r.utc()})
r.write_json(evidence / 'result.json', rec)
print(json.dumps(rec, ensure_ascii=False, indent=2), flush=True)
raise SystemExit(code)
