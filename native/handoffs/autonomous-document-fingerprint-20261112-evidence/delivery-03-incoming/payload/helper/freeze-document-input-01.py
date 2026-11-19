#!/usr/bin/env python3
"""Create-only metadata capture of one explicitly named box source group."""
import argparse
import datetime
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import traceback

LANE = Path('/home/user/levnote-dev-reliability-20261112-01')
HELPER = LANE / 'tools' / 'dev-remote-validation-20261112-03.py'
BASE = LANE / 'inputs' / 'reopen-red-01' / 'expected-manifest.json'
SOURCES = (LANE / 'work' / 'native', Path('/home/user/levnote-dev-drawio-20261112-01/source/native'), Path('/home/user/levnote-dev-experience-20261112-01/revision-02-source/native'), Path('/home/user/levnote-dev-experience-20261112-01/revision-03-source/native'), LANE / 'work/experience-tree-baseline-01/native', LANE / 'work/document-fingerprint-01/native')
HEAD = '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--label', required=True)
parser.add_argument('--source', required=True)
parser.add_argument('--allow', action='append', required=True)
args = parser.parse_args()
if not re.fullmatch(r'[a-z0-9][a-z0-9-]{0,79}', args.label):
    parser.error('fresh safe label required')
source = Path(args.source)
if source not in SOURCES or not source.is_absolute() or source.resolve() != source:
    parser.error('source must be one of the six coordinated box-native work roots')
os.umask(0o077)
sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('owned_runner02', HELPER)
r = importlib.util.module_from_spec(spec)
spec.loader.exec_module(r)
evidence = LANE / 'evidence' / ('freeze-' + args.label)
evidence.mkdir(parents=True, exist_ok=False)
summary = {'argv': sys.argv, 'cwd': str(Path.cwd()), 'pid': os.getpid(), 'started_at': r.utc(),
           'script': r.file_identity(__file__), 'runner': r.file_identity(HELPER),
           'source': str(source), 'allowed_changed_files': args.allow, 'head': HEAD, 'go_executed': False}
r.write_json(evidence / 'started.json', summary)
code = 2
try:
    if r.file_identity(BASE) != {'bytes': 199732, 'sha256': '7b35b91ed79f060550442270829e76f38eecf192c4e0d71ba96ef90bb8f5c06b'}:
        raise ValueError('the preserved baseline-plus-single-RED input changed')
    base = json.loads(BASE.read_bytes())
    if base['head'] != HEAD:
        raise ValueError('baseline head mismatch')
    baseline_files = dict(base['files'])
    del baseline_files['app/autonomous_reliability_test.go']
    before = r.inventory(source)
    r.write_json(evidence / 'source-before.json', before)
    names = set(baseline_files) | set(before['files'])
    changed = {name for name in names if baseline_files.get(name) != before['files'].get(name)}
    if changed != set(args.allow) or not set(baseline_files).issubset(before['files']):
        raise ValueError('source changes differ from the exact coordinated file group: ' + str(sorted(changed)))
    input_root = LANE / 'inputs' / args.label
    native = input_root / 'native'
    native.mkdir(parents=True, exist_ok=False)
    for name in before['files']:
        target = native / name
        target.parent.mkdir(parents=True, exist_ok=True)
        with target.open('xb') as f:
            f.write((source / name).read_bytes())
    if r.inventory(source) != before or r.inventory(native) != before:
        raise ValueError('source or copied bytes changed during metadata capture')
    overlay = [{'path': name, 'baseline': baseline_files.get(name), 'input': before['files'][name]} for name in sorted(changed)]
    expected = {'head': HEAD, **before, 'source_origin': str(source), 'owner': 'native-reliability/sole-product-lane',
                'label': args.label, 'overlay': overlay, 'input_count': len(before['files']),
                'input_bytes': sum(value['bytes'] for value in before['files'].values())}
    manifest = input_root / 'expected-manifest.json'
    r.write_json(manifest, expected)
    r.write_json(evidence / 'source-after.json', r.inventory(source))
    summary.update({'expected_manifest': {'path': str(manifest), **r.file_identity(manifest)},
                    'input_source': str(native), 'input_count': expected['input_count'], 'input_bytes': expected['input_bytes'],
                    'overlay': overlay, 'source_before': r.file_identity(evidence / 'source-before.json'),
                    'source_after': r.file_identity(evidence / 'source-after.json'), 'outcome': 'MATCHING_INPUT_CAPTURED_NOT_PRODUCT_ACCEPTANCE'})
    code = 0
except BaseException:
    text = traceback.format_exc()
    with (evidence / 'wrapper.stderr.txt').open('x', encoding='utf-8') as f:
        f.write(text)
    print(text, flush=True)
    summary['outcome'] = 'INPUT_CAPTURE_FAILED_NO_GO'
summary.update({'exit_code': code, 'ended_at': r.utc()})
r.write_json(evidence / 'result.json', summary)
with (evidence / 'wrapper.stdout.txt').open('x', encoding='utf-8') as f:
    f.write(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(summary, ensure_ascii=False, indent=2), flush=True)
raise SystemExit(code)
