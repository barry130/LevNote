#!/usr/bin/env python3
"""Two accuracy sentences only; source/evidence unchanged; create fresh primary."""
from pathlib import Path, PurePosixPath
import datetime
import hashlib
import io
import json
import os
import tarfile

lane = Path('/home/user/levnote-dev-reliability-20261112-01')
old_root = lane / 'delivery/document-fingerprint-review-02'
out = lane / 'delivery/document-fingerprint-review-03'
report = lane / 'work/document-fingerprint-01/native/handoffs/autonomous-document-fingerprint-20261112.md'
started = datetime.datetime.now(datetime.timezone.utc).isoformat()
os.umask(0o077)

def ident(data):
    return {'bytes': len(data), 'sha256': hashlib.sha256(data).hexdigest()}

def encoded(value):
    return (json.dumps(value, ensure_ascii=False, indent=2) + '\n').encode()

def put(path, data):
    path.parent.mkdir(exist_ok=True, parents=True)
    with path.open('xb') as stream:
        stream.write(data)
    assert path.read_bytes() == data

archive02 = old_root / 'document-fingerprint-review-02.tar.xz'
assert ident(archive02.read_bytes()) == {'bytes': 75652, 'sha256': '80041f649852ff172c33ef7eb200ecb639fdbc9687283386326db58dfc140ec1'}
old_report = report.read_bytes()
assert ident(old_report) == {'bytes': 26020, 'sha256': '10b458e0cbd2823aa1bef7510c8ec22c4ceee8070dc7331cc1b43d63dd19cba8'}
assert set(p.name for p in out.iterdir()) == {'report-accuracy-successor-03.py'}
changes = (
    ('当前已接受的可运行物理 Windows 产物仍是', '当前已接受的 runnable reference 仍是'),
    ('freeze、format、capture、test/vet 分开调用；每个 Go 命令均有原始', 'freeze、format、capture、test/vet 分开调用；每个已闭合 Go 命令均有原始'),
    ('文件名不是不存在的 `.intent.json`。', '文件名不是不存在的 `.intent.json`；中断 GREEN01 的数字/final/后清单缺失例外见第5节。'),
    ('delivery/document-fingerprint-review-02/bundle-manifest.json', 'delivery/document-fingerprint-review-03/bundle-manifest.json'),
)
new_report = old_report.decode()
for old, new in changes:
    assert new_report.count(old) == 1
    new_report = new_report.replace(old, new)
new_report = new_report.encode()
contents = {}
with tarfile.open(archive02, 'r:xz') as bundle:
    for member in bundle.getmembers():
        name = PurePosixPath(member.name)
        assert member.isfile() and not name.is_absolute() and '..' not in name.parts and '\\' not in member.name
        assert member.name not in contents
        contents[member.name] = bundle.extractfile(member).read()
manifest02 = json.loads(contents.pop('bundle-manifest.json'))
assert set(contents) == set(manifest02['files'])
assert all(ident(data) == manifest02['files'][name] for name, data in contents.items())
assert contents['report/autonomous-document-fingerprint-20261112.md'] == old_report
put(out / 'report-02-before-accuracy.md', old_report)
# Target has just been read and is the owned named report, not consumed product evidence.
assert report.read_bytes() == old_report
with report.open('wb') as stream:
    stream.write(new_report)
assert report.read_bytes() == new_report
contents['report/autonomous-document-fingerprint-20261112.md'] = new_report
contents['report/report-02-before-accuracy.md'] = old_report
contents['helper/report-accuracy-successor-03.py'] = Path(__file__).read_bytes()
refs = json.loads(contents['EvidenceReferences.json'])
refs['files']['report/autonomous-document-fingerprint-20261112.md'] = {'path': str(report), **ident(new_report), 'body_in_primary': True}
refs['files']['report/report-02-before-accuracy.md'] = {'path': str(out / 'report-02-before-accuracy.md'), **ident(old_report), 'body_in_primary': True}
refs['files']['helper/report-accuracy-successor-03.py'] = {'path': str(Path(__file__).resolve()), **ident(contents['helper/report-accuracy-successor-03.py']), 'body_in_primary': True}
refs['files']['prior-archive02'] = {'path': str(archive02), **ident(archive02.read_bytes()), 'body_in_primary': False}
refs['reference_count'] = len(refs['files'])
refs['body_in_primary_count'] = sum(v['body_in_primary'] for v in refs['files'].values())
contents['EvidenceReferences.json'] = encoded(refs)
manifest03 = dict(manifest02)
manifest03['label'] = 'document-fingerprint-review-03'
manifest03['report'] = {'path': str(report), **ident(new_report)}
manifest03['previous_report_and_archive_preserved'] = {'report': {'path': str(out / 'report-02-before-accuracy.md'), **ident(old_report)}, 'archive': {'path': str(archive02), **ident(archive02.read_bytes())}}
manifest03['report_accuracy_changes'] = [{'before': old, 'after': new} for old,new in changes]
manifest03['files'] = {name: ident(data) for name,data in sorted(contents.items())}
manifest03['origins'] = dict(manifest02['origins'])
manifest03['origins'].update({'report/report-02-before-accuracy.md': str(out / 'report-02-before-accuracy.md'), 'helper/report-accuracy-successor-03.py': str(Path(__file__).resolve())})
manifest_raw = encoded(manifest03)
put(out / 'bundle-manifest.json', manifest_raw)
contents['bundle-manifest.json'] = manifest_raw
archive = out / 'document-fingerprint-review-03.tar.xz'
with archive.open('xb') as stream:
    with tarfile.open(fileobj=stream, mode='w:xz', preset=9) as bundle:
        for name,data in sorted(contents.items()):
            member = tarfile.TarInfo(name)
            member.size = len(data); member.mode=0o600; member.mtime=0; member.uid=member.gid=0
            bundle.addfile(member, io.BytesIO(data))
assert archive.stat().st_size < 90000
with tarfile.open(archive, 'r:xz') as bundle:
    assert {member.name: bundle.extractfile(member).read() for member in bundle.getmembers()} == contents
assert ident(archive02.read_bytes()) == {'bytes': 75652, 'sha256': '80041f649852ff172c33ef7eb200ecb639fdbc9687283386326db58dfc140ec1'}
receipt = {'metadata_only': True, 'product_executed': False, 'started_at': started, 'ended_at': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'pid': os.getpid(), 'script': {'path': str(Path(__file__).resolve()), **ident(Path(__file__).read_bytes())}, 'report': {'path': str(report), **ident(new_report)}, 'archive': {'path': str(archive), **ident(archive.read_bytes())}, 'manifest': {'path': str(out / 'bundle-manifest.json'), **ident(manifest_raw)}, 'fixed_file_count': len(contents), 'fixed_file_bytes': sum(map(len,contents.values())), 'source_files': manifest03['source_pins'], 'source_file_count': 8, 'source_bytes': 52983, 'unchanged_source_and_original_product_receipts': True, 'all_members_readback_exact': True, 'previous02_preserved': manifest03['previous_report_and_archive_preserved'], 'exit_code': 0}
put(out / 'archive-receipt.json', encoded(receipt))
print(json.dumps(receipt, ensure_ascii=False, indent=2))
print('SMALL_PRIMARY03_ACCURACY_ONLY_READY_NO_GO')
