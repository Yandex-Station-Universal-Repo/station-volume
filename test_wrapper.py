"""Firmware guard fixture: a copied wrapper and a fake native executable only."""
from pathlib import Path
import subprocess
import tempfile

source = Path(__file__).resolve().with_name('station-volume.sh').read_text()
with tempfile.TemporaryDirectory(prefix='station-volume-guard-') as temporary:
    root = Path(temporary)
    (root / 'bin').mkdir()
    version = root / 'firmware'
    compatible = root / 'compatible'
    calls = root / 'calls'
    version.write_text('unverified-firmware')
    compatible.write_bytes(b'yandex,plum-evt2\0amlogic,axg\0')
    wrapper = root / 'station-volume.sh'
    # Test-only substitution, not a shipped environment variable or bypass.
    wrapper.write_text(
        source.replace('/system/vendor/quasar/version_full', str(version)).replace(
            '/proc/device-tree/compatible', str(compatible)
        )
    )
    helper = root / 'bin/native-volume'
    helper.write_text(f'#!/bin/sh\nprintf "%s\\n" "$*" >> "{calls}"\n[ "$1" != get ] || echo 65\n')
    helper.chmod(0o700)

    def run(*args):
        return subprocess.run(['sh', str(wrapper), *args], capture_output=True, text=True)

    for args in [('check-write',), ('set', '50'), ('set', '50', 'unexpected'),
                 ('check-write', 'extra'), ('arbitrary',)]:
        assert run(*args).returncode != 0
    assert not calls.exists(), 'firmware guard reached the native writer'
    assert run('get').stdout.strip() == '65'
    assert calls.read_text().splitlines() == ['get']

    version.write_text('0.369.1.50.4795647582.20261021.133')
    assert run('check-write').stdout.strip() == 'ok'
    assert calls.read_text().splitlines() == ['get'], 'preflight has a side effect'
    assert run('set', '50').returncode == 0
    assert calls.read_text().splitlines() == ['get', 'set 50']

print('STATION_VOLUME_READONLY_PREFLIGHT_AND_FIRMWARE_GUARD_PASS')
