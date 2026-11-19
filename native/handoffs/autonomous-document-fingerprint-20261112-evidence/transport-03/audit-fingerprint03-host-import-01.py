"""Prepared for Root review: independent fixed Windows pre/post import audit; only reads and stdout."""
import sys
sys.dont_write_bytecode = True
from pathlib import Path, PurePosixPath
from datetime import datetime, timezone
import hashlib
import io
import json
import os
import stat
import tarfile

HOST = Path('F:/AIproject/LevNote')
TRANSPORT = HOST / 'native/handoffs/autonomous-document-fingerprint-20261112-evidence/transport-03'
ARCHIVE = TRANSPORT / 'document-fingerprint-review-03.tar.xz'
INCOMING = HOST / 'native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming'
REPORT = HOST / 'native/handoffs/autonomous-document-fingerprint-20261112.md'
IMPORTER = TRANSPORT / 'import-fingerprint03-final.py'
AUDITOR = TRANSPORT / 'audit-fingerprint03-host-import-01.py'
ARCHIVE_PIN = (77196, 'fe2acbb546a93d35547bbdb95548b5a9e5f8ab6e6d403b04747fb6a0e0e90802')
MANIFEST_PIN = (133142, '9417143226b1dfe967106764a21d4b79e797eef22f3ec9d6a8fe17c4e41ed09d')
REPORT_PIN = (26084, '4344170197093c2230a6f1dcbd2f2850d9f442dfd2f28c807e7d34a0622b60a5')
REFS_PIN = (234740, 'bcaa1f2a9c812062486f783026c383b77ff9d4fb324f9f2fece66d28cff35b62')
IMPORTER_PIN = (11251, '77c89b41848ec420ee27a4813318f74cc86059839e4e5f465a8cd5698ae28d13')
HEAD = '8897244ffa4f00f5ec51f95545e7a3ca27d37d97'
SOURCE_ORIGIN = '/home/user/levnote-dev-reliability-20261112-01/inputs/document-final-core-tests-01/native'
REPORT_ORIGIN = '/home/user/levnote-dev-reliability-20261112-01/work/document-fingerprint-01/native/handoffs/autonomous-document-fingerprint-20261112.md'
SOURCE_PINS = {
    'core/document/document.go': ((12537, '6859017748c8884dd9cc84b185a405c0a78089aa0d84f191024489cb24c11219'), (14511, '44da6396913741a18ed5ebd1d0c668853d28ad399eba9b13abb02aa08f76a3fe')),
    'core/document/atomic.go': ((3516, '33d45a65253972d8547a5dc7acd1ac9fa04c0540ef11dd3de207676801a232cc'), (4873, '641601d61fe3da0ec83954c6b199cac46864099ffa78f09e2a309180491fec83')),
    'core/document/utf8_save_test.go': ((8197, '446f7542b0447fea4f29b92796f6cb4b32b3aeb249e914bfb279731f78302863'), (8238, '7c02ad0c795cadda2b93c443d6816d742ff13bf0e9ae26facb3e8149f31b3ec0')),
    'core/document/fingerprint_boundary_test.go': (None, (3399, 'f0d3204bf2b7430eb1a95ca183ade789816faf7752433cad06c2e6a74f434aef')),
    'core/document/fingerprint_public_test.go': (None, (2957, '3c94db4b1ac0604f03ebb3cc26fdf62e661dadfd266ad74b06d6d9fe587955d5')),
    'core/document/fingerprint_conflict_public_test.go': (None, (6993, 'c0556277c6d7ae7c7cf4a72d75d3901a23aedcedec94aa00711dc5ac0ab20b69')),
    'core/document/fingerprint_encoding_public_test.go': (None, (4870, '40212ed1a6c5319b16e2e107ae1aaa8530b3d381fe3dac85d64266436f974edc')),
    'core/document/fingerprint_directory_sync_public_linux_test.go': (None, (7142, '8f68c732927b6375edeb8b589facef356a360f944512b4564968afaa6fe51951')),
}
PROTECTED = {'native/handoffs/autonomous-reliability-20261112.md':
             (34988, '503dbf1c32d21865774379ebfeea419a9fa57ac77994cb2ddc51c6f2de9aaf27')}


def ident(body):
    return (len(body), hashlib.sha256(body).hexdigest())


def named(body):
    length, digest = ident(body)
    return {'bytes': length, 'sha256': digest}


def pin_named(pin):
    return {'bytes': pin[0], 'sha256': pin[1]}


def plain(path):
    assert path.is_absolute() and path.resolve().is_relative_to(HOST.resolve()), str(path)
    for node in [path, *path.parents]:
        if os.path.lexists(node):
            info = node.lstat()
            assert not stat.S_ISLNK(info.st_mode), 'symlink: ' + str(node)
            assert not (getattr(info, 'st_file_attributes', 0) & 0x400), 'reparse: ' + str(node)
    return path


def read(path):
    plain(path)
    assert stat.S_ISREG(path.lstat().st_mode), 'not regular: ' + str(path)
    return path.read_bytes()


def absent(path):
    plain(path)
    assert not os.path.lexists(path), 'expected absent: ' + str(path)


def pairs(rows):
    result = {}
    for key, value in rows:
        assert key not in result, 'duplicate JSON key: ' + key
        result[key] = value
    return result


def decode(body):
    return json.loads(body, object_pairs_hook=pairs)


def safe_name(name):
    parts = name.split('/')
    assert not PurePosixPath(name).is_absolute() and parts and all(p not in ('', '.', '..') for p in parts)
    assert not any(ch in name for ch in '\\:<>"|?*') and not any(p.endswith((' ', '.')) for p in parts), name
    reserved = {'CON', 'PRN', 'AUX', 'NUL', *('COM' + str(i) for i in range(1, 10)), *('LPT' + str(i) for i in range(1, 10))}
    assert not any(p.split('.')[0].upper() in reserved for p in parts), name
    return name


def inventory(root, expected_names):
    plain(root)
    assert root.is_dir()
    wanted_dirs = set()
    for name in expected_names:
        safe_name(name)
        wanted_dirs.update('/'.join(name.split('/')[:i]) for i in range(1, len(name.split('/'))))
    files, dirs_found = {}, set()
    for base, dirs, names in os.walk(root, followlinks=False):
        for name in dirs:
            p = plain(Path(base) / name)
            assert stat.S_ISDIR(p.lstat().st_mode)
            dirs_found.add(p.relative_to(root).as_posix())
        for name in names:
            p = Path(base) / name
            files[safe_name(p.relative_to(root).as_posix())] = named(read(p))
    assert set(files) == set(expected_names) and dirs_found == wanted_dirs, str(root)
    return dict(sorted(files.items()))


def canonical_pin(value):
    return named((json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(',', ':')) + '\n').encode('utf-8'))


def utc_value(value):
    parsed = datetime.fromisoformat(value)
    assert parsed.tzinfo is not None and parsed.utcoffset().total_seconds() == 0
    return parsed


assert os.name == 'nt' and len(sys.argv) == 1, 'fixed Windows read-only auditor, no arguments'
assert Path.cwd().resolve() == HOST.resolve() and Path(__file__).resolve() == AUDITOR.resolve()
started = datetime.now(timezone.utc)
archive = read(ARCHIVE)
assert ident(archive) == ARCHIVE_PIN
assert ident(read(IMPORTER)) == IMPORTER_PIN
contents, case_names = {}, set()
# This is an independent archive read. No receipt or importer code supplies member bodies.
with tarfile.open(fileobj=io.BytesIO(archive), mode='r:xz') as bundle:
    for member in bundle.getmembers():
        name = safe_name(member.name)
        assert member.type in (tarfile.REGTYPE, tarfile.AREGTYPE) and member.isfile() and member.sparse is None
        assert name not in contents and name.casefold() not in case_names and 0 <= member.size <= 400000
        stream = bundle.extractfile(member)
        assert stream is not None
        body = stream.read(member.size + 1)
        assert len(body) == member.size
        contents[name] = body
        case_names.add(name.casefold())
assert len(contents) == 269 and sum(len(v) for v in contents.values()) == 1025008
actual_members = {name: named(body) for name, body in sorted(contents.items())}
manifest_body = contents['bundle-manifest.json']
assert ident(manifest_body) == MANIFEST_PIN
manifest = decode(manifest_body)
assert manifest['metadata_only'] is True and manifest['product_executed'] is False
assert manifest['label'] == 'document-fingerprint-review-03' and manifest['task'] == 'task-11' and manifest['head'] == HEAD
assert manifest['source_origin'] == SOURCE_ORIGIN and manifest['source_total_bytes'] == 52983
assert set(contents) == set(manifest['files']) | {'bundle-manifest.json'} and len(manifest['files']) == 268
assert {name: pin for name, pin in actual_members.items() if name != 'bundle-manifest.json'} == manifest['files']
assert manifest['source_pins'] == {name: pin_named(final) for name, (_, final) in SOURCE_PINS.items()}
assert {name for name in contents if name.startswith('source/')} == {'source/' + name for name in SOURCE_PINS}
for name, (old, final) in SOURCE_PINS.items():
    assert ident(contents['source/' + name]) == final
    if old is not None:
        assert ident(contents['baseline/' + name]) == old
report_body = contents['report/autonomous-document-fingerprint-20261112.md']
assert ident(report_body) == REPORT_PIN and manifest['report'] == {'path': REPORT_ORIGIN, **pin_named(REPORT_PIN)}
assert manifest['not_whole_application_or_windows_acceptance'] is True
refs_body = contents['EvidenceReferences.json']
assert ident(refs_body) == REFS_PIN
refs = decode(refs_body)
assert refs['metadata_only'] is True and refs['product_executed'] is False and refs['not_a_complete_backup'] is True
assert refs['reference_count'] == len(refs['files']) == 659
present, remote_only = 0, 0
for name, ref in refs['files'].items():
    assert set(ref) == {'path', 'bytes', 'sha256', 'body_in_primary'} and type(ref['body_in_primary']) is bool
    assert isinstance(ref['path'], str) and ref['path'].startswith('/home/user/')
    if ref['body_in_primary']:
        assert name in contents and {key: ref[key] for key in ('bytes', 'sha256')} == actual_members[name]
        present += 1
    else:
        assert name not in contents
        remote_only += 1
assert present == refs['body_in_primary_count'] == 264 and remote_only == 395
for name in SOURCE_PINS:
    assert refs['files']['source/' + name] == {'path': SOURCE_ORIGIN + '/' + name, **actual_members['source/' + name], 'body_in_primary': True}
post = os.path.lexists(INCOMING)
source_observed, expected_rows, before_bodies, backups_expected = {}, [], {}, {}
for name, (old, final) in SOURCE_PINS.items():
    target = HOST / 'native' / name
    absent(target.with_name(target.name + '.mirror-fingerprint03.tmp'))
    expected_rows.append({'name': name, 'path': str(target), 'before': None if old is None else pin_named(old),
                          'after': pin_named(final), 'changed': True, 'created': old is None})
    if post:
        body = read(target)
        assert body == contents['source/' + name] and ident(body) == final
        source_observed[name] = named(body)
        before_bodies[target] = body
    elif old is None:
        absent(target)
        source_observed[name] = None
        before_bodies[target] = None
    else:
        body = read(target)
        assert body == contents['baseline/' + name] and ident(body) == old
        source_observed[name] = named(body)
        before_bodies[target] = body
    if old is not None:
        backups_expected['native/' + name] = pin_named(old)
protected_observed = {}
for name, pin in PROTECTED.items():
    target = HOST / name
    body = read(target)
    assert ident(body) == pin
    protected_observed[name] = named(body)
    before_bodies[target] = body
    backups_expected[name] = pin_named(pin)
assert len(backups_expected) == 4
receipts, incoming_files = {}, None
if not post:
    absent(INCOMING)
    absent(REPORT)
else:
    plain(INCOMING)
    assert INCOMING.is_dir() and sorted(p.name for p in INCOMING.iterdir()) == ['backup-receipt.json', 'backups', 'document-fingerprint-review-03.tar.xz', 'import-receipt.json', 'payload']
    assert read(INCOMING / 'document-fingerprint-review-03.tar.xz') == archive
    actual_payload = inventory(INCOMING / 'payload', contents)
    assert actual_payload == actual_members and len(actual_payload) == 269 and sum(p['bytes'] for p in actual_payload.values()) == 1025008
    actual_backups = inventory(INCOMING / 'backups', backups_expected)
    assert actual_backups == dict(sorted(backups_expected.items()))
    for name, (old, _) in SOURCE_PINS.items():
        if old is not None:
            assert read(INCOMING / 'backups/native' / name) == contents['baseline/' + name]
    for name in PROTECTED:
        assert read(INCOMING / 'backups' / name) == before_bodies[HOST / name]
    assert read(REPORT) == report_body
    backup_raw = read(INCOMING / 'backup-receipt.json')
    import_raw = read(INCOMING / 'import-receipt.json')
    backup_receipt, import_receipt = decode(backup_raw), decode(import_raw)
    expected_backups = [{'original': str(HOST / name), 'backup': str(INCOMING / 'backups' / name), **pin}
                        for name, pin in backups_expected.items()]
    assert backup_receipt['status'] == 'TASK11_PREMIRROR_EXACT_ORIGINAL_BACKUPS_AND_NEW_TARGET_ABSENCE'
    assert backup_receipt['archive'] == pin_named(ARCHIVE_PIN) and backup_receipt['backups'] == expected_backups
    assert backup_receipt['plannedMirrors'] == expected_rows and backup_receipt['newReport'] == str(REPORT) and backup_receipt['productCommands'] == 0
    assert import_receipt['status'] == 'TASK11_PRIMARY03_EIGHT_BYTE_EXACT_SOURCE_MIRRORS_AND_REPORT_NOT_APP_ACCEPTANCE'
    assert type(import_receipt['pid']) is int and import_receipt['pid'] > 0
    assert len(import_receipt['argv']) == 1 and Path(import_receipt['argv'][0]).resolve() == IMPORTER.resolve()
    assert Path(import_receipt['cwd']).resolve() == HOST.resolve() and Path(import_receipt['python']).resolve() == Path(sys.executable).resolve()
    assert import_receipt['script'] == {'path': str(IMPORTER.resolve()), **pin_named(IMPORTER_PIN)}
    assert import_receipt['archive'] == {'path': str(ARCHIVE), **pin_named(ARCHIVE_PIN)} and import_receipt['manifest'] == pin_named(MANIFEST_PIN)
    assert import_receipt['incoming'] == str(INCOMING) and import_receipt['entryCount'] == 269 and import_receipt['payloadBytes'] == 1025008
    assert import_receipt['allEntriesByteExact'] is True and import_receipt['allFinalEightByteExact'] is True
    assert import_receipt['finalEight'] == expected_rows and import_receipt['backups'] == expected_backups
    assert import_receipt['newReport'] == {'path': str(REPORT), **pin_named(REPORT_PIN)}
    assert import_receipt['protectedReportsUnchanged'] == list(PROTECTED) and import_receipt['EvidenceReferences'] == pin_named(REFS_PIN)
    assert import_receipt['originalExpectedAndMapsReferencedNotCompleteBackup'] is True and import_receipt['productCommands'] == 0
    assert import_receipt['notWindowsOrWholeApplicationAcceptance'] is True
    assert utc_value(import_receipt['startedUTC']) <= utc_value(backup_receipt['createdUTC']) <= utc_value(import_receipt['endedUTC']) <= datetime.now(timezone.utc)
    receipts = {'backup': {'path': str(INCOMING / 'backup-receipt.json'), **named(backup_raw), 'actual_body': backup_receipt},
                'import': {'path': str(INCOMING / 'import-receipt.json'), **named(import_raw), 'actual_body': import_receipt}}
    expected_incoming = {'payload/' + name: pin for name, pin in actual_members.items()}
    expected_incoming.update({'backups/' + name: pin for name, pin in backups_expected.items()})
    expected_incoming.update({'document-fingerprint-review-03.tar.xz': pin_named(ARCHIVE_PIN),
                              'backup-receipt.json': named(backup_raw), 'import-receipt.json': named(import_raw)})
    incoming_files = inventory(INCOMING, expected_incoming)
    assert incoming_files == dict(sorted(expected_incoming.items())) and len(incoming_files) == 276
# Guard the complete inspected state again. No writes, imports, subprocesses or product commands occurred.
assert read(ARCHIVE) == archive and ident(read(IMPORTER)) == IMPORTER_PIN
for path, body in before_bodies.items():
    if body is None:
        absent(path)
    else:
        assert read(path) == body
for name in SOURCE_PINS:
    target = HOST / 'native' / name
    absent(target.with_name(target.name + '.mirror-fingerprint03.tmp'))
if post:
    assert read(REPORT) == report_body and inventory(INCOMING, incoming_files) == incoming_files
else:
    absent(INCOMING)
    absent(REPORT)
ended = datetime.now(timezone.utc)
result = {'status': 'INDEPENDENT_PRIMARY03_POSTMIRROR_AUDIT_OK_NOT_PRODUCT_ACCEPTANCE' if post else 'INDEPENDENT_PRIMARY03_PREMIRROR_AUDIT_OK_NO_IMPORT_EXECUTED',
          'metadata_only': True, 'file_writes': 0, 'spawns': 0, 'product_commands': 0, 'pid': os.getpid(), 'cwd': str(Path.cwd()),
          'started_at': started.isoformat(), 'ended_at': ended.isoformat(), 'elapsed_s': (ended - started).total_seconds(),
          'helper': {'path': str(AUDITOR), **named(read(AUDITOR))}, 'archive': {'path': str(ARCHIVE), **pin_named(ARCHIVE_PIN)},
          'actual_archive_regular_unique_safe_members': 269, 'actual_archive_unpacked_bytes': 1025008,
          'actual_all_268_bodies_hashed_independently_match_manifest': True, 'manifest': pin_named(MANIFEST_PIN),
          'actual_269_member_inventory': canonical_pin(actual_members), 'actual_source_eight_bytes': 52983,
          'actual_source_eight_match_fixed_final_pins_in_archive': True, 'actual_report_match_fixed_pin_in_archive': pin_named(REPORT_PIN),
          'actual_refs_match_fixed_pin': pin_named(REFS_PIN), 'refs_count': 659, 'actual_primary_ref_bodies_verified': present,
          'remote_only_refs_not_backed_up': remote_only, 'original_expected_and_maps_referenced_not_complete_backup': True,
          'postmirror': post, 'source_eight_current': source_observed, 'protected_reports_current': protected_observed,
          'original_three_baselines_match_fixed_pins_in_archive': True, 'expected_source_mirrors': expected_rows,
          'actual_incoming_files': None if incoming_files is None else {'count': 276, 'inventory_pin': canonical_pin(incoming_files)},
          'actual_payload_269_and_backups_four_exact': post, 'actual_new_report_byte_exact': post, 'receipts': receipts,
          'inspected_state_before_after_unchanged': True, 'all_mirror_temporary_paths_absent': True,
          'not_windows_or_whole_application_acceptance': True}
print(json.dumps(result, ensure_ascii=False, indent=2), flush=True)
