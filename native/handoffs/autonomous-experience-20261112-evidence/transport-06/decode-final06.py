"""Decode pinned opaque transport parts, create archive only; never import source."""
from pathlib import Path
import base64
import hashlib
import json
import os
import stat
import sys
from datetime import datetime, timezone

ROOT = Path('F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/transport-06')
EXPECTED = (85709, '25b935ae97a020bed1552c4fb5468130dabd8a884a6080826bfde8d91e288621')
PINS = [
    (12150, 'ada79f9ba7addf5ed6f3a856f37b770539ac1193224b0f0163fd5c062a2ab3c3'),
    (12150, '74c4ede6075f3c47d44cf05f7c306e5317c84dc57a8b0e529e1731c44b868b39'),
    (12150, 'f5b92d0b0e0db787c3455f5525531938f3286fc02025df4c03630097dd6eaed5'),
    (12150, 'a8937a62da2abd8d0c870abe5862856cdc77bf3edb7ba671924f0410fd67fa31'),
    (12150, '8f6fa83662cdc23f3e3dc47e2e743570e0d92b884b301170640e34d21a7ad631'),
    (12150, '85fce1d2bfe95c367dfabf7460536e6c476d7e11722f1978868e305a3e14ed78'),
    (12150, 'b23b3de744144b944ec7c9e0a38067ea4d48df1d8935f2200c58b3bd449dd2e8'),
    (12150, '8d6eb8b3cd12ada3e0a7ac076dbf1a93443693bc41845f1d0976046399647c39'),
    (12150, 'c95d5d103af1b9854f2e60410cb41d8fae0ddf639501f0682091e3969f2ffe3d'),
    (6359, '2c91b3491f456e68d59176303882a8de7b9fcfc835f65921805a81ef849d3e53'),
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
    body = plain(ROOT / ('final06-review.part%02d.b64' % index)).read_bytes()
    assert identity(body) == pin
    lines = body.splitlines()
    assert all(len(line) == 80 for line in lines[:-1]) and 0 < len(lines[-1]) <= 80
    encoded = b''.join(lines)
    assert len(encoded) == (6280 if index == 10 else 12000)
    parts.append(encoded)
encoded = b''.join(parts)
assert len(encoded) == 114280
archive = base64.b64decode(encoded, validate=True)
assert identity(archive) == EXPECTED
output = plain(ROOT / 'final06-review.tar.gz')
assert not output.exists(), 'create-only archive target exists'
with output.open('xb') as stream:
    stream.write(archive)
    stream.flush()
    os.fsync(stream.fileno())
assert output.read_bytes() == archive
receipt = {'status': 'FINAL06_ARCHIVE_DECODED_BYTE_EXACT_NO_IMPORT', 'startedUTC': started,
           'endedUTC': datetime.now(timezone.utc).isoformat(), 'pid': os.getpid(),
           'archive': {'path': str(output), 'bytes': len(archive), 'sha256': EXPECTED[1]},
           'encodedCharacters': 114280, 'partCount': 10,
           'metadataOnly': True, 'sourceWrites': 0, 'productCommands': 0}
receipt_path = plain(ROOT / 'decoder-receipt.json')
body = (json.dumps(receipt, indent=2) + '\n').encode('utf-8')
with receipt_path.open('xb') as stream:
    stream.write(body)
    stream.flush()
    os.fsync(stream.fileno())
assert receipt_path.read_bytes() == body
print(json.dumps({'receipt': {'path': str(receipt_path), 'bytes': len(body), 'sha256': identity(body)[1]}, **receipt}))
