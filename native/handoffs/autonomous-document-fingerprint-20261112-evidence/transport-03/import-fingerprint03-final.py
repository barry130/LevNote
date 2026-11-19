"""Prepared for Root review: metadata-only exact primary03 and eight sources; no Go/acceptance commands."""
from pathlib import Path, PurePosixPath
from datetime import datetime, timezone
import hashlib
import io
import json
import os
import stat
import sys
import tarfile

HOST = Path('F:/AIproject/LevNote')
TRANSPORT = HOST / 'native/handoffs/autonomous-document-fingerprint-20261112-evidence/transport-03'
ARCHIVE = TRANSPORT / 'document-fingerprint-review-03.tar.xz'
DEST = HOST / 'native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming'
REPORT = HOST / 'native/handoffs/autonomous-document-fingerprint-20261112.md'
EXPECTED_ARCHIVE = (77196, 'fe2acbb546a93d35547bbdb95548b5a9e5f8ab6e6d403b04747fb6a0e0e90802')
EXPECTED_MANIFEST = (133142, '9417143226b1dfe967106764a21d4b79e797eef22f3ec9d6a8fe17c4e41ed09d')
EXPECTED_REPORT = (26084, '4344170197093c2230a6f1dcbd2f2850d9f442dfd2f28c807e7d34a0622b60a5')
EXPECTED_REFS = (234740, 'bcaa1f2a9c812062486f783026c383b77ff9d4fb324f9f2fece66d28cff35b62')
SOURCE_ORIGIN = '/home/user/levnote-dev-reliability-20261112-01/inputs/document-final-core-tests-01/native'
REPORT_ORIGIN = '/home/user/levnote-dev-reliability-20261112-01/work/document-fingerprint-01/native/handoffs/autonomous-document-fingerprint-20261112.md'
PINS = {
    'core/document/document.go': ((12537, '6859017748c8884dd9cc84b185a405c0a78089aa0d84f191024489cb24c11219'), (14511, '44da6396913741a18ed5ebd1d0c668853d28ad399eba9b13abb02aa08f76a3fe')),
    'core/document/atomic.go': ((3516, '33d45a65253972d8547a5dc7acd1ac9fa04c0540ef11dd3de207676801a232cc'), (4873, '641601d61fe3da0ec83954c6b199cac46864099ffa78f09e2a309180491fec83')),
    'core/document/utf8_save_test.go': ((8197, '446f7542b0447fea4f29b92796f6cb4b32b3aeb249e914bfb279731f78302863'), (8238, '7c02ad0c795cadda2b93c443d6816d742ff13bf0e9ae26facb3e8149f31b3ec0')),
    'core/document/fingerprint_boundary_test.go': (None, (3399, 'f0d3204bf2b7430eb1a95ca183ade789816faf7752433cad06c2e6a74f434aef')),
    'core/document/fingerprint_public_test.go': (None, (2957, '3c94db4b1ac0604f03ebb3cc26fdf62e661dadfd266ad74b06d6d9fe587955d5')),
    'core/document/fingerprint_conflict_public_test.go': (None, (6993, 'c0556277c6d7ae7c7cf4a72d75d3901a23aedcedec94aa00711dc5ac0ab20b69')),
    'core/document/fingerprint_encoding_public_test.go': (None, (4870, '40212ed1a6c5319b16e2e107ae1aaa8530b3d381fe3dac85d64266436f974edc')),
    'core/document/fingerprint_directory_sync_public_linux_test.go': (None, (7142, '8f68c732927b6375edeb8b589facef356a360f944512b4564968afaa6fe51951')),
}
PROTECTED = {
    'native/handoffs/autonomous-reliability-20261112.md': (34988, '503dbf1c32d21865774379ebfeea419a9fa57ac77994cb2ddc51c6f2de9aaf27'),
}


def utc():
    return datetime.now(timezone.utc).isoformat()


def identity(body):
    return (len(body), hashlib.sha256(body).hexdigest())


def named_id(body):
    n, h = identity(body)
    return {'bytes': n, 'sha256': h}


def pin_id(pin):
    return {'bytes': pin[0], 'sha256': pin[1]}


def plain(path):
    for node in [path, *path.parents]:
        if node.exists() or node.is_symlink():
            info = node.lstat()
            assert not stat.S_ISLNK(info.st_mode), 'symlink: ' + str(node)
            assert not (getattr(info, 'st_file_attributes', 0) & 0x400), 'reparse point: ' + str(node)
    assert path.resolve().is_relative_to(HOST.resolve()) or path == HOST
    return path


def fresh_write(path, body):
    plain(path)
    assert not path.exists(), 'create-only target exists: ' + str(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    plain(path.parent)
    with path.open('xb') as stream:
        stream.write(body)
        stream.flush()
        os.fsync(stream.fileno())
    assert path.read_bytes() == body


def require_before(target, before):
    plain(target)
    if before is None:
        assert not target.exists(), 'new source appeared: ' + str(target)
    else:
        assert target.is_file() and target.read_bytes() == before, 'source changed: ' + str(target)


assert os.name == 'nt' and len(sys.argv) == 1, 'fixed Windows metadata-only import, no arguments'
assert Path.cwd().resolve() == HOST.resolve(), 'run from the named workspace'
started = utc()
archive = plain(ARCHIVE).read_bytes()
assert identity(archive) == EXPECTED_ARCHIVE
assert not plain(DEST).exists() and not plain(REPORT).exists(), 'create-only incoming and report required'
contents = {}
with tarfile.open(fileobj=io.BytesIO(archive), mode='r:xz') as bundle:
    for member in bundle.getmembers():
        parts = member.name.split('/')
        name = PurePosixPath(member.name)
        assert member.isfile() and not name.is_absolute()
        assert not any(p in ('', '.', '..') for p in parts)
        assert '\\' not in member.name and ':' not in member.name
        assert member.name not in contents and 0 <= member.size <= 400000
        body = bundle.extractfile(member).read()
        assert len(body) == member.size
        contents[member.name] = body
assert len(contents) == 269 and sum(map(len, contents.values())) == 1025008
manifest_body = contents['bundle-manifest.json']
assert identity(manifest_body) == EXPECTED_MANIFEST
manifest = json.loads(manifest_body)
assert manifest['metadata_only'] is True and manifest['product_executed'] is False
assert manifest['label'] == 'document-fingerprint-review-03' and manifest['task'] == 'task-11'
assert manifest['head'] == '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
assert manifest['source_origin'] == SOURCE_ORIGIN and manifest['source_total_bytes'] == 52983
assert set(contents) == set(manifest['files']) | {'bundle-manifest.json'}
for name, pin in manifest['files'].items():
    assert named_id(contents[name]) == pin, 'member identity: ' + name
assert manifest['source_pins'] == {name: pin_id(final_pin) for name, (_, final_pin) in PINS.items()}
assert {name for name in contents if name.startswith('source/')} == {'source/' + name for name in PINS}
report = contents['report/autonomous-document-fingerprint-20261112.md']
assert identity(report) == EXPECTED_REPORT
assert manifest['report'] == {'path': REPORT_ORIGIN, **pin_id(EXPECTED_REPORT)}
assert identity(contents['EvidenceReferences.json']) == EXPECTED_REFS
refs = json.loads(contents['EvidenceReferences.json'])
for name in PINS:
    assert refs['files']['source/' + name] == {'path': SOURCE_ORIGIN + '/' + name, **named_id(contents['source/' + name]), 'body_in_primary': True}
assert manifest['not_whole_application_or_windows_acceptance'] is True
# Original expected/maps bodies deliberately remain on box, fully pinned by references.
assert any(v['body_in_primary'] is False and v['path'].startswith('/home/user/') for v in refs['files'].values())

# Complete source, incoming, new report, temporary and protected-report preflight before first write.
originals, rows = {}, []
for name, (old_pin, final_pin) in PINS.items():
    new_body = contents['source/' + name]
    assert identity(new_body) == final_pin
    target = plain(HOST / 'native' / name)
    if old_pin is None:
        assert not target.exists(), 'expected new source absent: ' + name
        before = None
    else:
        assert target.is_file(), 'expected existing source: ' + name
        before = target.read_bytes()
        assert identity(before) == old_pin, 'exact observed old source mismatch: ' + name
        assert contents['baseline/' + name] == before
    temporary = plain(target.with_name(target.name + '.mirror-fingerprint03.tmp'))
    assert not temporary.exists()
    originals[target] = before
    rows.append({'name': name, 'path': str(target), 'before': None if before is None else named_id(before), 'after': named_id(new_body), 'changed': True, 'created': before is None})
protected_before = {}
for name, pin in PROTECTED.items():
    target = plain(HOST / name)
    before = target.read_bytes()
    assert identity(before) == pin, 'protected report drift: ' + name
    protected_before[target] = before
assert not DEST.exists() and not REPORT.exists()
DEST.mkdir()
fresh_write(DEST / 'document-fingerprint-review-03.tar.xz', archive)
for name, body in contents.items():
    fresh_write(DEST / 'payload' / Path(*PurePosixPath(name).parts), body)
backups = []
for target, before in {**originals, **protected_before}.items():
    if before is None:
        continue
    backup = DEST / 'backups' / target.relative_to(HOST)
    fresh_write(backup, before)
    backups.append({'original': str(target), 'backup': str(backup), **named_id(before)})
backup_receipt = {'status': 'TASK11_PREMIRROR_EXACT_ORIGINAL_BACKUPS_AND_NEW_TARGET_ABSENCE', 'createdUTC': utc(), 'archive': named_id(archive), 'backups': backups, 'plannedMirrors': rows, 'newReport': str(REPORT), 'productCommands': 0}
fresh_write(DEST / 'backup-receipt.json', (json.dumps(backup_receipt, indent=2) + '\n').encode())
for target, before in originals.items():
    require_before(target, before)
for target, before in protected_before.items():
    assert plain(target).read_bytes() == before
assert not plain(REPORT).exists()
assert plain(ARCHIVE).read_bytes() == archive
for row in rows:
    target = plain(Path(row['path']))
    before, new_body = originals[target], contents['source/' + row['name']]
    require_before(target, before)
    if before is None:
        fresh_write(target, new_body)
    else:
        temp = plain(target.with_name(target.name + '.mirror-fingerprint03.tmp'))
        fresh_write(temp, new_body)
        require_before(target, before)
        assert temp.read_bytes() == new_body
        os.replace(temp, target)
    assert target.read_bytes() == new_body
fresh_write(REPORT, report)
for target, before in protected_before.items():
    assert plain(target).read_bytes() == before
for row in rows:
    assert named_id(plain(Path(row['path'])).read_bytes()) == row['after']
for name, body in contents.items():
    assert (DEST / 'payload' / Path(*PurePosixPath(name).parts)).read_bytes() == body
for backup in backups:
    assert named_id(plain(Path(backup['backup'])).read_bytes()) == {k: backup[k] for k in ('bytes', 'sha256')}
assert plain(ARCHIVE).read_bytes() == archive
receipt = {'status': 'TASK11_PRIMARY03_EIGHT_BYTE_EXACT_SOURCE_MIRRORS_AND_REPORT_NOT_APP_ACCEPTANCE', 'startedUTC': started, 'endedUTC': utc(), 'pid': os.getpid(), 'argv': sys.argv, 'cwd': str(Path.cwd()), 'python': sys.executable, 'script': {'path': str(Path(__file__).resolve()), **named_id(Path(__file__).read_bytes())}, 'archive': {'path': str(ARCHIVE), **named_id(archive)}, 'manifest': named_id(manifest_body), 'incoming': str(DEST), 'entryCount': 269, 'payloadBytes': 1025008, 'allEntriesByteExact': True, 'allFinalEightByteExact': True, 'finalEight': rows, 'backups': backups, 'newReport': {'path': str(REPORT), **named_id(report)}, 'protectedReportsUnchanged': list(PROTECTED), 'EvidenceReferences': named_id(contents['EvidenceReferences.json']), 'originalExpectedAndMapsReferencedNotCompleteBackup': True, 'productCommands': 0, 'notWindowsOrWholeApplicationAcceptance': True}
fresh_write(DEST / 'import-receipt.json', (json.dumps(receipt, indent=2) + '\n').encode())
print(json.dumps({'receipt': {'path': str(DEST / 'import-receipt.json'), **named_id((DEST / 'import-receipt.json').read_bytes())}, **receipt}, indent=2))
