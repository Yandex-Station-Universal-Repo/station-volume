from pathlib import Path
import subprocess, tempfile

source = Path(__file__).resolve().with_name('station-volume.sh').read_text()
with tempfile.TemporaryDirectory(prefix='station-volume-guard-') as temporary:
    root = Path(temporary)
    (root / 'bin').mkdir()
    fw = root / 'fw'
    dt = root / 'dt'
    calls = root / 'calls'
    wrapper = root / 'station-volume.sh'
    wrapper.write_text(
        source.replace('/system/vendor/quasar/version_full', str(fw)).replace(
            '/proc/device-tree/compatible', str(dt)
        )
    )
    helper = root / 'bin/native-volume'
    helper.write_text(f'#!/bin/sh\necho "$*" >> "{calls}"\n[ "$1" != get ] || echo 30\n')
    helper.chmod(0o700)

    def run(*args):
        return subprocess.run(['sh', str(wrapper), *args], capture_output=True, text=True)

    def allowed(fw_text, dt_bytes):
        fw.write_text(fw_text)
        dt.write_bytes(dt_bytes)
        return run('check-write').returncode == 0 and run('check-write').stdout.strip() == 'ok'

    models = [
        # Previously verified Midi build (commercial Midi is yandex,cucumber),
        # then the qualified Mini2 build.
        ('0.369.1.46.4778262955.20261013.160', b'yandex,cucumber\0amlogic,axg\0'),
        ('0.369.1.50.4794659041.20261021.299', b'yandex,station-mini2\0amlogic,axg\0'),
        # Lite2 .187 and .196, exact builds and their own compatible strings.
        ('0.369.1.50.4795647582.20261021.133', b'yandex,plum-evt2\0yandex,plum\0amlogic,axg\0'),
        ('0.369.1.50.4794654278.20261021.133', b'yandex,prune-evt2\0yandex,prune\0amlogic,axg\0'),
        # Station2 .251 (shared Midi device-tree identity, exact build).
        ('0.369.1.50.4794649087.20261021.295', b'yandex,station-midi\0amlogic,axg\0'),
    ]
    for fw_text, dt_bytes in models:
        assert allowed(fw_text, dt_bytes), fw_text
        # The same build must not be accepted on a different compatible board.
        assert not allowed(fw_text, b'yandex,unknown-board\0amlogic,axg\0'), fw_text
    assert not calls.exists(), 'check-write must never reach the helper'

    fw.write_text('0.369.1.50.4795647582.20261021.133')
    dt.write_bytes(b'yandex,plum-evt2\0yandex,plum\0amlogic,axg\0')
    assert run('set', '25').returncode == 0
    assert calls.read_text().splitlines() == ['set 25']

    # An unqualified build must never reach the helper.
    fw.write_text('future-firmware')
    assert run('check-write').returncode != 0
    assert run('set', '25').returncode != 0
    assert run('set', '25', '--force').returncode != 0
    assert run('arbitrary').returncode != 0
    assert calls.read_text().splitlines() == ['set 25']

print('EXACT_FW_DT_GUARD_FOR_MINI2_LITE2_STATION2_AND_NO_FORCE_PASS')
