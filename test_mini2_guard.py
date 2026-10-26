from pathlib import Path
import subprocess,tempfile
source=Path(__file__).resolve().with_name('station-volume.sh').read_text()
with tempfile.TemporaryDirectory(prefix='mini2-guard-') as temporary:
    root=Path(temporary);(root/'bin').mkdir();fw=root/'fw';dt=root/'dt';calls=root/'calls'
    wrapper=root/'station-volume.sh'
    wrapper.write_text(source.replace('/system/vendor/quasar/version_full',str(fw)).replace('/proc/device-tree/compatible',str(dt)))
    helper=root/'bin/native-volume';helper.write_text(f'#!/bin/sh\necho "$*" >> "{calls}"\n[ "$1" != get ] || echo 30\n');helper.chmod(0o700)
    def run(*args):return subprocess.run(['sh',str(wrapper),*args],capture_output=True,text=True)
    fw.write_text('0.369.1.50.4794659041.20261021.299');dt.write_bytes(b'yandex,station-mini2\0amlogic,axg\0')
    assert run('check-write').stdout.strip()=='ok' and not calls.exists()
    assert run('set','25').returncode==0 and calls.read_text().splitlines()==['set 25']
    dt.write_bytes(b'yandex,cucumber\0');assert run('check-write').returncode!=0 and run('set','25').returncode!=0
    dt.write_bytes(b'yandex,station-mini2\0');fw.write_text('future-firmware');assert run('check-write').returncode!=0
    fw.write_text('0.369.1.50.4794659041.20261021.299')
    assert run('set','25','--force').returncode!=0 and run('arbitrary').returncode!=0
    assert calls.read_text().splitlines()==['set 25']
print('MINI2_EXACT_FW_DT_GUARD_AND_NO_FORCE_PASS')
