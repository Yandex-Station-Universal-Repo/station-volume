#!/usr/bin/env python3
"""Package the unchanged guarded native-volume helper for a YAM v2 target."""
import argparse
import json
from pathlib import Path
import re
import shutil
import subprocess
import tomllib

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--namespace', required=True)
    p.add_argument('--platform', required=True, choices=['yandexmini_2', 'yandexplum', 'yandexstation_2'])
    p.add_argument('--native-volume', type=Path, required=True, help='AArch64 static helper built from this source')
    p.add_argument('--publisher', type=Path, required=True)
    p.add_argument('--profiles', type=Path, required=True)
    p.add_argument('--work-dir', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    a = p.parse_args()
    root = Path(__file__).resolve().parent.parent
    if not re.fullmatch(r'[a-z0-9-]{1,64}', a.namespace): p.error('invalid namespace')
    if a.work_dir.resolve().is_relative_to(root): p.error('staging must be outside the checkout')
    if a.output.exists(): p.error('output already exists')
    binary = a.native_volume.read_bytes()
    if binary[:5] != b'\x7fELF\x02' or binary[18:20] != b'\xb7\x00': p.error('expected AArch64 helper')
    a.work_dir.mkdir(parents=True, exist_ok=False)
    payload = a.work_dir/'payload'; (payload/'bin').mkdir(parents=True)
    shutil.copyfile(a.native_volume, payload/'bin/native-volume')
    (payload/'bin/native-volume').chmod(0o755)
    for name in ['station-volume.sh', 'LICENSE']:
        shutil.copyfile(root/name, payload/name)
    (payload/'station-volume.sh').chmod(0o755)
    text = (root/'plugin.toml').read_text()
    text, count = re.subn(r'^id\s*=.*$', 'id = '+json.dumps(a.namespace+'.volume'), text, flags=re.M)
    if count != 1: raise RuntimeError('unexpected descriptor id')
    descriptor = tomllib.loads(text)
    assert descriptor['version'] == '1.0.2+all4.20261026'
    assert descriptor['launch']['interpreter'] == 'sh'
    (payload/'plugin.toml').write_text(text)
    manifest = {'name': a.namespace+'/volume', 'version': descriptor['version'], 'license': 'MIT',
        'description': 'Guarded Quasar native volume, unchanged firmware checks',
        'targets': [a.platform], 'files': {'share/luna/plugins/'+a.namespace+'.volume/': 'payload/'}}
    (a.work_dir/'yam.yaml').write_text(json.dumps(manifest, indent=2)+'\n')
    a.output.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run([str(a.publisher.resolve()), 'pack', '--source', str(a.work_dir.resolve()), '--profiles', str(a.profiles.resolve()), '--platform', a.platform, '--output', str(a.output.resolve())], check=True)
    print('CANDIDATE_ONLY', a.output)

if __name__ == '__main__': main()
