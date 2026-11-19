"""Decode pinned opaque transport parts, create archive only; never import source."""
from pathlib import Path
import base64
import hashlib
import json
import os
import stat
import sys
from datetime import datetime, timezone

ROOT = Path('F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/transport-03')
EXPECTED = (77196, 'fe2acbb546a93d35547bbdb95548b5a9e5f8ab6e6d403b04747fb6a0e0e90802')
PINS = [
    (12150, '68f8ca742bb564943966c795d1d1af852e5c58c3274d399f350d4e96eb342fb2'),
    (12150, 'a9d7464d5411abad45dc587168e7cc83f4399d23bd41555430639726c689fb37'),
    (12150, '93ad4d5d9e5e8896108b31091a0c1d1fb00daee162b40cd53fdd36aea3acba32'),
    (12150, 'e69d74919b3a129bea581d92089331b790a04c1a7ef35a5f747a73d373dcb111'),
    (12150, 'fdb446619417a21b8194be7f77cb04747bf091c15520f2dab9e116d9484edd35'),
    (12150, '4313fc30297f2f92dcd026f1a3e93b81e92e72afde4252160b7cc11df2440238'),
    (12150, 'ea60cbd06c65f335e3b80d0d4f7dd5c3e4f3789378171aeec2c3a4355a2468f6'),
    (12150, '2b556a9c3ecf81d883bec42c0a30cc2c95621c1d62bf0be207af0ead45770081'),
    (7015, 'c92c8f3b9d0285c59d97ed6e21daa4624a1d27ad6e3aa3a648617b8fd9bd971d'),
]


def identity(body):
    return (len(body), hashlib.sha256(body).hexdigest())


def plain(path):
    for node in [path, *path.parents]:
        if node.exists() or node.is_symlink():
            info = node.lstat()
            assert not stat.S_ISLNK(info.st_mode) and not (getattr(info, 'st_file_attributes', 0) & 0x400)
    assert path.resolve().is_relative_to(ROOT.resolve())
    return path


assert os.name == 'nt' and len(sys.argv) == 1
assert Path.cwd().resolve() == Path('F:/AIproject/LevNote').resolve()
started = datetime.now(timezone.utc).isoformat()
assert not plain(ROOT / 'decoder-receipt.json').exists()
parts = []
for index, pin in enumerate(PINS, 1):
    body = plain(ROOT / ('fingerprint-review03.part%02d.b64' % index)).read_bytes()
    assert identity(body) == pin
    lines = body.splitlines()
    assert all(len(line) == 80 for line in lines[:-1]) and 0 < len(lines[-1]) <= 80
    encoded = b''.join(lines)
    assert len(encoded) == (6928 if index == 9 else 12000)
    parts.append(encoded)
encoded = b''.join(parts)
assert len(encoded) == 102928
archive = base64.b64decode(encoded, validate=True)
assert identity(archive) == EXPECTED
output = plain(ROOT / 'document-fingerprint-review-03.tar.xz')
assert not output.exists(), 'create-only archive target exists'
with output.open('xb') as stream:
    stream.write(archive)
    stream.flush()
    os.fsync(stream.fileno())
assert output.read_bytes() == archive
receipt = {'status': 'DOCUMENT_FINGERPRINT03_ARCHIVE_DECODED_BYTE_EXACT_NO_IMPORT', 'startedUTC': started,
           'endedUTC': datetime.now(timezone.utc).isoformat(), 'pid': os.getpid(),
           'archive': {'path': str(output), 'bytes': len(archive), 'sha256': EXPECTED[1]},
           'encodedCharacters': 102928, 'partCount': 9,
           'metadataOnly': True, 'sourceWrites': 0, 'productCommands': 0}
receipt_path = plain(ROOT / 'decoder-receipt.json')
body = (json.dumps(receipt, indent=2) + '\n').encode('utf-8')
with receipt_path.open('xb') as stream:
    stream.write(body)
    stream.flush()
    os.fsync(stream.fileno())
assert receipt_path.read_bytes() == body
print(json.dumps({'receipt': {'path': str(receipt_path), 'bytes': len(body), 'sha256': identity(body)[1]}, **receipt}))
