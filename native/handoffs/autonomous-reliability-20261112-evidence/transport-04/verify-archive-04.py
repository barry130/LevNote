"""Metadata-only verification of opaque box-authored archive transport."""
from pathlib import Path
import base64
import hashlib
import json

ROOT = Path('F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112-evidence/transport-04')
parts = [ROOT / ('payload-%03d.b64' % i) for i in (1, 2, 3)]
encoded = b''.join(p.read_bytes() for p in parts)
data = base64.b64decode(b''.join(encoded.split()), validate=True)
actual = {'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}
print(json.dumps({'parts': [{'name': p.name, 'bytes': p.stat().st_size, 'sha256': hashlib.sha256(p.read_bytes()).hexdigest()} for p in parts], 'archive': actual}))
assert actual == {'bytes': 28976, 'sha256': '3a580b8dc6b23bdbec8e10258bd00783953f74be71ff77a1a4946082db1a55e3'}, 'opaque transport differs from box archive; no import permitted'
with (ROOT / 'review-01-primary.tar.xz').open('xb') as stream:
    stream.write(data)
print('VERIFIED_ARCHIVE_WRITTEN_NOT_PRODUCT_ACCEPTANCE')
