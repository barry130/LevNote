"""Import verified, opaque box evidence; no product code or commands execute."""
from pathlib import Path, PurePosixPath
import base64
import datetime
import hashlib
import json
import os
import sys
import tarfile

ROOT = Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112-evidence/transport-04')
DEST = Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112-evidence/review-01')
REPORT = Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112.md')
ARCHIVE = ROOT / 'review-01-primary.tar.xz'
EXPECTED_ARCHIVE = {'bytes': 28976, 'sha256': '3a580b8dc6b23bdbec8e10258bd00783953f74be71ff77a1a4946082db1a55e3'}
EXPECTED_PAYLOAD = {'bytes': 39023, 'sha256': '8cd042d708100dcdaa85b645756a859848d7f1197eb10c8d556a4a14af4bb537'}
EXPECTED_OLD = {'bytes': 14017, 'sha256': '678ba7a178818ecf63a0f7097e120d92cd59000394eff10c96eaed45487cf69e'}
EXPECTED_NEW = {'bytes': 34988, 'sha256': '503dbf1c32d21865774379ebfeea419a9fa57ac77994cb2ddc51c6f2de9aaf27'}

def utc():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()

def identity(data):
    return {'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}

started = utc()
encoded = b''.join((ROOT / ('payload-%03d.b64' % i)).read_bytes() for i in (1, 2, 3))
assert identity(encoded) == EXPECTED_PAYLOAD
archive_bytes = ARCHIVE.read_bytes()
assert identity(archive_bytes) == EXPECTED_ARCHIVE
assert base64.b64decode(b''.join(encoded.split()), validate=True) == archive_bytes
assert not REPORT.is_symlink() and REPORT.resolve() == REPORT
old_report = REPORT.read_bytes()
assert identity(old_report) == EXPECTED_OLD, 'named report drifted; no overwrite permitted'
assert not DEST.exists(), 'fresh create-only destination required'
contents = {}
with tarfile.open(ARCHIVE, 'r:xz') as bundle:
    for member in bundle.getmembers():
        path = PurePosixPath(member.name)
        assert member.isfile() and not path.is_absolute() and '..' not in path.parts and '\\' not in member.name
        assert member.name not in contents
        data = bundle.extractfile(member).read()
        assert len(data) == member.size
        contents[member.name] = data
assert len(contents) == 82 and sum(map(len, contents.values())) == 290242
manifest_raw = contents['bundle-manifest.json']
assert identity(manifest_raw) == {'bytes': 14781, 'sha256': '52d5e03a76fc2d689c2c8d2537f8a3ceabe88f4d8f5cf9c689b585483f05b37a'}
manifest = json.loads(manifest_raw)
assert manifest['metadata_only'] and not manifest['product_executed']
assert set(contents) == set(manifest['files']) | {'bundle-manifest.json'}
for name, expected in manifest['files'].items():
    assert identity(contents[name]) == expected, name
new_report = contents['report/autonomous-reliability-20261112.md']
assert identity(new_report) == EXPECTED_NEW
assert contents['report/incoming-report.md'] == old_report == new_report[:14017]
allowed_runs = {'drawio-incident-green-02', 'drawio-ui-green-01', 'drawio-ui-green-02', 'drawio-ui-red-01', 'experience-list-affected-01', 'experience-tree-current-green-02', 'native-path-affected-01', 'native-path-vet-01'}
assert {PurePosixPath(name).parts[1] for name in contents if name.startswith('runs/')} == allowed_runs
DEST.mkdir()
for name, data in contents.items():
    target = DEST.joinpath(*PurePosixPath(name).parts)
    assert target.resolve().is_relative_to(DEST.resolve())
    target.parent.mkdir(parents=True, exist_ok=True)
    with target.open('xb') as stream:
        stream.write(data)
    assert target.read_bytes() == data
assert REPORT.read_bytes() == old_report
# The exact absolute replacement destination and source are checked before move.
replacement = REPORT.with_name(REPORT.name + '.mirror04.tmp')
assert REPORT.resolve() == Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112.md')
assert replacement.resolve() == Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112.md.mirror04.tmp')
with replacement.open('xb') as stream:
    stream.write(new_report)
    stream.flush()
    os.fsync(stream.fileno())
assert identity(replacement.read_bytes()) == EXPECTED_NEW and REPORT.read_bytes() == old_report
os.replace(replacement, REPORT)
assert REPORT.read_bytes() == new_report
receipt = {
    'metadata_only': True, 'product_executed': False,
    'not_whole_application_or_windows_acceptance': True,
    'started_at': started, 'ended_at': utc(), 'pid': os.getpid(),
    'argv': sys.argv, 'cwd': str(Path.cwd()), 'python': sys.executable,
    'script': {'path': str(Path(__file__).resolve()), **identity(Path(__file__).read_bytes())},
    'box_archive': {'path': '/home/user/levnote-dev-reliability-20261112-01/evidence/reliability-review-export-04/review-01-primary.tar.xz', **EXPECTED_ARCHIVE},
    'original_complete_box_bundle': manifest['original_complete_box_bundle'],
    'joined_payload': EXPECTED_PAYLOAD,
    'import_root': str(DEST), 'file_count': len(contents), 'file_bytes': sum(map(len, contents.values())),
    'all_manifest_entries_verified': True,
    'all_imported_files': {name: identity(data) for name, data in sorted(contents.items())},
    'report': {'path': str(REPORT), 'before': EXPECTED_OLD, 'after': EXPECTED_NEW, 'unchanged_prefix_bytes': 14017, 'preserved_original': str(DEST / 'report/incoming-report.md')},
    'scope': manifest['host_scope'],
    'g1_failure_unchanged_g2_separate_successor': True,
    'exit_code': 0,
}
with (ROOT / 'import-receipt.json').open('x', encoding='utf-8', newline='\n') as stream:
    json.dump(receipt, stream, ensure_ascii=False, indent=2)
    stream.write('\n')
print(json.dumps({key: value for key, value in receipt.items() if key != 'all_imported_files'}, ensure_ascii=False, indent=2))
print('ALL_82_FILES_IMPORTED_BYTE_EXACT_NOT_PRODUCT_ACCEPTANCE')
