from pathlib import Path
import collections, hashlib, json, os, subprocess, sys, time
root=Path('F:/AIproject/LevNote')
phase, attempt=sys.argv[1:3]
selectors={'format':['./formats/drawio/'], 'view':['./modules/graphview/'], 'app':['./app/','-run','Drawio|Diagram|Graph'], 'test':['./app/','./cmd/...','./core/...','./formats/...','./modules/...','./ui/...']}
if phase not in (*selectors,'vet','build') or not attempt.isdigit(): raise SystemExit('invalid phase/attempt')
out=root/f'.localtools/flowix-chrome-20261116-{phase}-{attempt}'
out.mkdir(exist_ok=False)
def identities():
 paths=[]
 for name in ('app','cmd','core','formats','modules','ui'):
  paths.extend(p for p in (root/'native'/name).rglob('*') if p.is_file())
 paths.extend(root/p for p in ('native/go.mod','native/go.sum','native/vendor/modules.txt','.localtools/gate.sh'))
 return {p.relative_to(root).as_posix():hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(paths)}
before=identities()
def scoped_inputs(values):
 if phase != 'format': return values
 return {k:v for k,v in values.items() if k.startswith('native/formats/drawio/') or k in ('native/go.mod','native/go.sum','native/vendor/modules.txt','.localtools/gate.sh')}
scoped_before=scoped_inputs(before)
(out/'inputs-before.json').write_text(json.dumps(before,indent=2),encoding='utf-8')
(out/'scoped-inputs-before.json').write_text(json.dumps(scoped_before,indent=2),encoding='utf-8')
bash='C:/Users/canace/.dsh/dsh-ops/shells/niubash-prepared/e3765d439d5d873f50e910706e380ece9fa29d6349c02cf383cb54aeae4e4701/winuxcmd/usr/bin/bash.exe'
args=[bash,str(root/'.localtools/gate.sh')]
if phase in selectors: args+=['test','-json','-count=1','-timeout=300s']+selectors[phase]
else: args+=[phase]
env=os.environ.copy()
env['LEVNOTE_WORKBENCH_SCREENSHOT_DIR']=str(out/'images')
env['LEVNOTE_PALETTE_SCREENSHOT_DIR']=str(out/'images')
env['LEVNOTE_MARKDOWN_INLINE_SCREENSHOT_DIR']=str(out/'images')
env['LEVNOTE_GRAPH_STYLE_SCREENSHOT_DIR']=str(out/'images')
env['LEVNOTE_DRAWIO_MULTI_SCREENSHOT_DIR']=str(out/'images')
start=time.time()
with (out/'stdout.jsonl').open('wb') as stdout, (out/'stderr.log').open('wb') as stderr:
 p=subprocess.run(args,cwd=root,env=env,stdout=stdout,stderr=stderr,timeout=600)
after=identities()
scoped_after=scoped_inputs(after)
(out/'inputs-after.json').write_text(json.dumps(after,indent=2),encoding='utf-8')
(out/'scoped-inputs-after.json').write_text(json.dumps(scoped_after,indent=2),encoding='utf-8')
counts=collections.Counter(); tests={}; failed=[]; run=[]
for line in (out/'stdout.jsonl').read_text(encoding='utf-8',errors='replace').splitlines():
 try: e=json.loads(line)
 except ValueError: continue
 if e.get('Test'):
  identity=e['Package']+'/'+e['Test']; action=e.get('Action')
  if action=='run': run.append(identity)
  if action in ('pass','fail','skip'): tests[identity]=action; counts[action]+=1
  if action=='fail': failed.append(identity)
result={'phase':phase,'args':args,'exitCode':p.returncode,'elapsedSeconds':round(time.time()-start,3),'inputCount':len(before),'inputsSame':before==after,'changedInputs':sorted(k for k in before.keys()|after.keys() if before.get(k)!=after.get(k)),'scopedInputCount':len(scoped_before),'scopedInputsSame':scoped_before==scoped_after,'scopedChangedInputs':sorted(k for k in scoped_before.keys()|scoped_after.keys() if scoped_before.get(k)!=scoped_after.get(k)),'runCount':len(run),'counts':dict(counts),'failed':failed,'directory':str(out)}
(out/'tests.json').write_text(json.dumps(tests,indent=2),encoding='utf-8')
(out/'result.json').write_text(json.dumps(result,indent=2),encoding='utf-8')
print(json.dumps(result,ensure_ascii=False))
if p.returncode:
 print((out/'stderr.log').read_text(encoding='utf-8',errors='replace')[-6000:])
 print((out/'stdout.jsonl').read_text(encoding='utf-8',errors='replace')[-10000:])
sys.exit(p.returncode)
