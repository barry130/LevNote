"""Metadata-only archive inventory; expose the untouched manifest for reading."""
from pathlib import Path, PurePosixPath
import hashlib
import json
import tarfile

ROOT = Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112-evidence/transport-04')
archive = ROOT / 'review-01-primary.tar.xz'
data = archive.read_bytes()
assert len(data) == 28976 and hashlib.sha256(data).hexdigest() == '3a580b8dc6b23bdbec8e10258bd00783953f74be71ff77a1a4946082db1a55e3'
with tarfile.open(archive, 'r:xz') as bundle:
    members = bundle.getmembers()
    names = set()
    for member in members:
        path = PurePosixPath(member.name)
        assert member.isfile() and not path.is_absolute() and '..' not in path.parts and '\\' not in member.name
        assert member.name not in names
        names.add(member.name)
    manifests = [member for member in members if PurePosixPath(member.name).name == 'bundle-manifest.json']
    assert len(manifests) == 1
    raw_manifest = bundle.extractfile(manifests[0]).read()
    with (ROOT / 'bundle-manifest.json').open('xb') as out:
        out.write(raw_manifest)
    print(json.dumps({'archive': {'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}, 'member_count': len(members), 'member_bytes': sum(member.size for member in members), 'members': [{'path': member.name, 'bytes': member.size} for member in members], 'manifest': {'member_path': manifests[0].name, 'bytes': len(raw_manifest), 'sha256': hashlib.sha256(raw_manifest).hexdigest()}, 'product_executed': False}, indent=2))
