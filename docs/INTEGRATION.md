# Integration preflight (`check-write`)

`station-volume.sh check-write` validates the current model/firmware pair against
the existing allowlist without opening Quasar IPC, writing volume or changing any
state. It prints `ok` and exits 0 only for a verified build on the matching board;
otherwise it prints a diagnostic and exits non-zero.

Integrations such as the AirPlay shared-controls option call the package launcher
and never execute `bin/native-volume` directly. The launcher accepts exactly `get`,
`watch`, `check-write` and `set PERCENT`; unknown commands and extra arguments are
rejected. `set` performs the same preflight before invoking the helper, and
`--force` no longer exists.

This is not a new native-volume engine and not a second Quasar protocol. The
existing `volume_set` directive, quantisation and guards are unchanged; this
proposal only adds the read-only preflight, strict argument validation and tests.

## Verified builds — 26 October 2026

Each entry requires both the exact firmware build and its own device-tree
compatible string. A qualified build on a different board is rejected before the
helper runs.

| Model | Firmware | DT compatible | Qualification |
|---|---|---|---|
| Midi (previous build) | `0.369.1.46.4778262955.20261013.160` | `yandex,cucumber` | existing verified build, board check added |
| Mini 2 | `0.369.1.50.4794659041.20261021.299` | `yandex,station-mini2` | read + `30 → 25 → 30` |
| Lite 2 Plum | `0.369.1.50.4795647582.20261021.133` | `yandex,plum-evt2` | read + `80 → 60 → 80` |
| Lite 2 Prune | `0.369.1.50.4794654278.20261021.133` | `yandex,prune-evt2` | read + `70 → 50 → 70` |
| Station 2 | `0.369.1.50.4794649087.20261021.295` | `yandex,station-midi` | read + `75 → 55 → 75` |

The round trips read the quantised value back and restored the original level. No
`--force`, Midi spoofing or inferred compatibility was used. Other models and
firmware builds remain rejected; do not widen the list from mocks.

## Tests

```sh
go vet ./...
go test -race -count=1 ./...
python3 test_wrapper.py
python3 test_mini2_guard.py
sh -n station-volume.sh
./build.sh
```

`test_mini2_guard.py` covers all five accepted builds, wrong-board rejection,
unqualified firmware, argument validation, absence of `--force` and the fact that
`check-write` never reaches the helper. `test_wrapper.py` covers the copied-wrapper
path with a fake helper. Generated `.yap` archives include `LICENSE`.

## Limits

- Qualification is limited to the exact builds above; it is not a general
  firmware-range claim.
- `check-write` proves the guard, not that a future Quasar build will accept writes.
- The station-volume package still performs the actual IPC write; this proposal
  does not add a second volume implementation.
