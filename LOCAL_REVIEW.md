# LOCAL PR draft: reusable station-volume preflight and regression tests

Base: Yandex-Station-Universal-Repo/station-volume62d1e17. Not published.

## Reuse, not a second native-volume implementation

AirPlay calls the existing package launcher for get/set and respects its model,
feature and firmware guards. It never executes bin/native-volume directly and
never passes --force. No Quasar volume protocol is copied into YAM or AirPlay.

This small station-volume proposal adds:

- `station-volume.sh check-write`: reads the existing exact firmware allowlist,
  returns `ok` only for a verified build, never opens IPC or changes volume.
- Strict launcher argument validation (unknown commands/extra flags rejected).
- An injectable internal `runWith(args, configPath, writer)` for isolated tests,
  not a new user-configurable firmware bypass or alternative volume engine.
- Golden typed VolumeSet packet, plain/wrapped state/mute, malformed/nonfinite
  inputs, loopback native acknowledgement/quantisation and no-write-before-probe.
- A copied-wrapper fixture tests unknown/known firmware and read-only preflight.
- Include the existing MIT license in generated .yap distributions.

Existing model= yandexmidi and the verified firmware list are deliberately
unchanged. These tests do NOT qualify Mini2/Lite2/Station2 firmware writes.
AirPlay integration on those devices remains blocked by those real guards until
separate owner-approved native read/write tests and a reviewed compatibility PR.
Never claim compatibility by spoofing Midi or enabling --force.

Local tests: go test -race ./..., go vet ./..., sh -n station-volume.sh,
python3 test_wrapper.py; build.sh creates the ARM64 package.
No device changes, live logs, sound, public push or PR are involved.
