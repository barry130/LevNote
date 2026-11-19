"""Import pinned box bytes into a new host archive and exact allowlisted mirrors; no Go."""
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
TRANSPORT = HOST / 'native/handoffs/autonomous-experience-20261112-evidence/transport-06'
ARCHIVE = TRANSPORT / 'final06-review.tar.gz'
DEST = HOST / 'native/handoffs/autonomous-experience-20261112-evidence/delivery-06-incoming'
REPORT = HOST / 'native/handoffs/autonomous-experience-final06-20261113.md'
PREFIX = 'native/handoffs/autonomous-experience-20261112-evidence/delivery-06/'
MANIFEST_NAME = PREFIX + 'transport-inputs.json'
EXPECTED_ARCHIVE = (85709, '25b935ae97a020bed1552c4fb5468130dabd8a884a6080826bfde8d91e288621')
EXPECTED_MANIFEST = (37274, 'ac267abf0777dd81eaf7bef2f9cc64fa862e3850909719f0df706972553d4cdf')
EXPECTED_REPORT = (21483, '320979a87b66032f88b33d467ce0f44eb837c59ad868edec58528cec3e1611c8')
# Every source has an explicit observed old identity and an immutable final identity.
PINS = {
    'app.go': ((39471, 'b4e8b96ac45a336d3cc2524f513fe652f6cd638c075ad53fb590c509d09fdbde'), (39471, 'b4e8b96ac45a336d3cc2524f513fe652f6cd638c075ad53fb590c509d09fdbde')),
    'view.go': ((24252, 'a102bd5266bcf35296ba5973c2856a8211b1118ef7068f3f2470502a0d8fb935'), (24375, '8b702c15dfcd265821ec10fe7a8f52927f6ffe667a349ff04b8b281c67ba3b77')),
    'workspace.go': ((8220, '3a2863c49554a4574d45d650c5d388c33d64599f428241366c700baed2e50be9'), (8339, 'd9f458f1cb4d597932b58161f546f1e43874aa5eeb9bf25eb9c47ca40519177a')),
    'outline.go': ((15407, '7caed4b54a69af411f916b42b1e5d5a9de7db7fb9480799c38d3552d2299bf3a'), (15523, 'c75435237f5d9eb9b466ed8b755cc8c22818f5dc4a6a57ac2045ae7136cad8cf')),
    'list_lifetime.go': ((2345, '14c4d6cc436a6633ca5055fb18b6bc5b63703ebce0d7c8a817092ea5bdf24bb5'), (2345, '14c4d6cc436a6633ca5055fb18b6bc5b63703ebce0d7c8a817092ea5bdf24bb5')),
    'list_lifetime_test.go': ((17378, '982e3ec121e9d5c020df3e9d4661748e894186cf91976de118d32be47da120a2'), (17378, '982e3ec121e9d5c020df3e9d4661748e894186cf91976de118d32be47da120a2')),
    'list_handle_lifetime_test.go': ((7900, 'ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5'), (7900, 'ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5')),
    'acceptance_test.go': ((25229, '0d54cd2308b545d98cb3f2d81b35ef367c310b66658e2b3ea4672dd7edd6e362'), (25229, '0d54cd2308b545d98cb3f2d81b35ef367c310b66658e2b3ea4672dd7edd6e362')),
    'workers.go': ((8114, 'd6dbc465014b4e7dd1ff4fbc26f3064abf42542682069461c2f2c37411f0eab6'), (8225, '73c65d6aedc8b517c808314aae439f38c1c4137b013c2ca2db885e1fc88d9b83')),
}
PROTECTED = {
    'native/handoffs/autonomous-experience-20261112.md': (19105, '5f16c8c368b043475bc3fab36217bbbd72de8748ce0751918d704033edfe55aa'),
    'native/handoffs/autonomous-reliability-20261112.md': (34988, '503dbf1c32d21865774379ebfeea419a9fa57ac77994cb2ddc51c6f2de9aaf27'),
}


def utc():
    return datetime.now(timezone.utc).isoformat()


def identity(body):
    return (len(body), hashlib.sha256(body).hexdigest())


def named_id(body):
    size, sha = identity(body)
    return {'bytes': size, 'sha256': sha}


def plain(path):
    # Refuse symlinks/junctions/reparse points at every existing ancestor.
    for node in [path, *path.parents]:
        if node.exists() or node.is_symlink():
            info = node.lstat()
            assert not stat.S_ISLNK(info.st_mode)
            assert not (getattr(info, 'st_file_attributes', 0) & 0x400)
    assert path.resolve().is_relative_to(HOST.resolve()) or path == HOST
    return path


def fresh_write(path, body):
    plain(path)
    assert not path.exists()
    path.parent.mkdir(parents=True, exist_ok=True)
    plain(path.parent)
    with path.open('xb') as stream:
        stream.write(body)
        stream.flush()
        os.fsync(stream.fileno())
    assert path.read_bytes() == body


assert os.name == 'nt' and len(sys.argv) == 1, 'fixed Windows host metadata importer only'
assert Path.cwd().resolve() == HOST.resolve(), 'run from the named workspace'
started = utc()
plain(ARCHIVE)
archive = ARCHIVE.read_bytes()
assert identity(archive) == EXPECTED_ARCHIVE
assert not plain(DEST).exists() and not plain(REPORT).exists(), 'create-only destination required'
contents = {}
with tarfile.open(fileobj=io.BytesIO(archive), mode='r:gz') as tar:
    for member in tar.getmembers():
        parts = member.name.split('/')
        posix = PurePosixPath(member.name)
        assert member.isfile() and member.name.startswith(PREFIX)
        assert not posix.is_absolute() and not any(x in ('', '.', '..') for x in parts)
        assert '\\' not in member.name and ':' not in member.name
        assert member.name not in contents and 0 <= member.size <= 400000
        body = tar.extractfile(member).read()
        assert len(body) == member.size
        contents[member.name] = body
assert len(contents) == 52 and sum(map(len, contents.values())) == 375017
manifest_body = contents[MANIFEST_NAME]
assert identity(manifest_body) == EXPECTED_MANIFEST
manifest = json.loads(manifest_body)
assert manifest['authorProductCommands'] == 0 and manifest['notWindowsOrWholeApplicationAcceptance'] is True
assert manifest['baseline'] == '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
assert manifest['sourceRoot'] == '/home/user/levnote-dev-experience-20261112-01/revision-06-source'
records = {e['path']: e for e in manifest['entries']}
assert len(records) == 51 and set(contents) == set(records) | {MANIFEST_NAME}
for name, record in records.items():
    assert named_id(contents[name]) == {key: record[key] for key in ['bytes', 'sha256']}
assert len(manifest['finalNine']) == 9
assert {e['path'] for e in manifest['finalNine']} == {PREFIX + 'final/native/app/' + name for name in PINS}
report = contents[PREFIX + 'report-06.md']
assert identity(report) == EXPECTED_REPORT and manifest['report'] == records[PREFIX + 'report-06.md']

# Complete destination/source/protected-report preflight before the first write.
originals, mirror_rows = {}, []
for name, (old_pin, final_pin) in PINS.items():
    logical = PREFIX + 'final/native/app/' + name
    new_body = contents[logical]
    assert identity(new_body) == final_pin
    target = plain(HOST / 'native/app' / name)
    assert target.is_file()
    before = target.read_bytes()
    assert identity(before) in [old_pin, final_pin], 'unapproved prior source: ' + name
    temporary = plain(target.with_name(target.name + '.mirror-final06.tmp'))
    assert not temporary.exists()
    originals[target] = before
    mirror_rows.append({'name': name, 'path': str(target), 'before': named_id(before),
                        'after': named_id(new_body), 'changed': before != new_body})
for relative, expected in PROTECTED.items():
    path = plain(HOST / relative)
    originals[path] = path.read_bytes()
    assert identity(originals[path]) == expected, 'protected old report drifted'
assert not DEST.exists() and not REPORT.exists()
DEST.mkdir()
fresh_write(DEST / 'final06-review.tar.gz', archive)
for name, body in contents.items():
    fresh_write(DEST / 'payload' / Path(*PurePosixPath(name).parts), body)
# Backup receipt is complete and persisted before any source replacement.
backup_rows = []
for path, body in originals.items():
    relative = path.relative_to(HOST)
    backup = DEST / 'backups' / relative
    fresh_write(backup, body)
    backup_rows.append({'original': str(path), 'backup': str(backup), **named_id(body)})
backup_receipt = {'status': 'PREMIRROR_EXACT_ORIGINAL_BACKUPS', 'createdUTC': utc(),
                  'archive': named_id(archive), 'backups': backup_rows,
                  'plannedMirrors': mirror_rows, 'newReport': str(REPORT), 'productCommands': 0}
fresh_write(DEST / 'backup-receipt.json', (json.dumps(backup_receipt, indent=2) + '\n').encode('utf-8'))
for path, before in originals.items():
    assert plain(path).read_bytes() == before, 'destination changed after preflight'
assert not plain(REPORT).exists()
for row in mirror_rows:
    target = plain(Path(row['path']))
    new_body = contents[PREFIX + 'final/native/app/' + row['name']]
    assert target.read_bytes() == originals[target], 'source changed before exact mirror'
    if row['changed']:
        temp = plain(target.with_name(target.name + '.mirror-final06.tmp'))
        fresh_write(temp, new_body)
        assert target.read_bytes() == originals[target] and temp.read_bytes() == new_body
        os.replace(temp, target)
    assert target.read_bytes() == new_body
fresh_write(REPORT, report)
for relative, expected in PROTECTED.items():
    assert identity(plain(HOST / relative).read_bytes()) == expected
for row in mirror_rows:
    assert named_id(plain(Path(row['path'])).read_bytes()) == row['after']
for name, body in contents.items():
    assert (DEST / 'payload' / Path(*PurePosixPath(name).parts)).read_bytes() == body
receipt = {
    'status': 'FINAL06_OPAQUE_ARCHIVE_AND_NINE_MIRRORS_COMPLETE_NOT_APP_ACCEPTANCE',
    'startedUTC': started, 'endedUTC': utc(), 'pid': os.getpid(), 'argv': sys.argv,
    'cwd': str(Path.cwd()), 'python': sys.executable,
    'script': {'path': str(Path(__file__).resolve()), **named_id(Path(__file__).read_bytes())},
    'archive': {'path': str(ARCHIVE), **named_id(archive)}, 'manifest': named_id(manifest_body),
    'incoming': str(DEST), 'entryCount': len(contents), 'payloadBytes': sum(map(len, contents.values())),
    'allEntriesByteExact': True, 'allFinalNineByteExact': True, 'finalNine': mirror_rows,
    'backups': backup_rows, 'newReport': {'path': str(REPORT), **named_id(report)},
    'protectedReportsUnchanged': list(PROTECTED), 'productCommands': 0,
    'notWindowsOrWholeApplicationAcceptance': True,
}
fresh_write(DEST / 'import-receipt.json', (json.dumps(receipt, indent=2) + '\n').encode('utf-8'))
print(json.dumps({'receipt': {'path': str(DEST / 'import-receipt.json'), **named_id((DEST / 'import-receipt.json').read_bytes())}, **receipt}, indent=2))
