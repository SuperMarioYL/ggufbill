"""Emit and inspect the data of one manifest using a metadata-only GGUF fixture."""
import hashlib,json,subprocess,tempfile
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
fixture=ROOT/'internal/gguf/testdata/qwen3-27b-q4_k_m.gguf'
with tempfile.TemporaryDirectory(prefix='ggufbill-demo-') as directory:
    binary=str(Path(directory)/'ggufbill');output=Path(directory)/'demo.ggufbill.json'
    subprocess.run(['go','build','-o',binary,'./cmd/ggufbill'],cwd=ROOT,check=True)
    emitted=subprocess.run([binary,'emit',str(fixture),'--non-interactive','--context','4096','--kv','f16','--operator','demo-operator','--output',str(output)],capture_output=True,text=True,check=True)
    manifest=json.loads(output.read_text())
    result={'manifest_version':manifest['manifest_version'],'model_name':manifest['model']['name'],
            'fixture_bytes':fixture.stat().st_size,'sha256_matches':manifest['model']['sha256']==hashlib.sha256(fixture.read_bytes()).hexdigest(),
            'context_length':manifest['runtime']['context_length'],'kv_cache_dtype':manifest['runtime']['kv_cache_dtype'],
            'operator':manifest['audit']['operator']}
    print(json.dumps(result,indent=2))
    assert result['sha256_matches'] and result['context_length']==4096
