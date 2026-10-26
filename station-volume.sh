#!/bin/sh
# Existing upstream native-volume helper; exact verified model/firmware guards.
# Mini2: typed30->25->30; Lite2: 80->60->80 and 70->50->70; Station2:
# 75->55->75 acknowledgement/restoration verified 26.10.2026. Other
# firmware/models are NOT inferred compatible from these tests.
set -eu
here=$(dirname "$0")
check_write() {
 build=$(cat /system/vendor/quasar/version_full 2>/dev/null || true)
 compatible=$(tr '\000' ' ' < /proc/device-tree/compatible 2>/dev/null || true)
 case "$build" in
  # Previously verified Midi build; the commercial Midi reports yandex,cucumber.
  0.369.1.46.4778262955.20261013.160)
   case " $compatible " in *" yandex,cucumber "*) return 0;; esac
   ;;
  0.369.1.50.4794659041.20261021.299)
   case " $compatible " in *" yandex,station-mini2 "*) return 0;; esac
   ;;
  # Lite 2: typed VolumeSet 80->60->80 / 70->50->70 acknowledged 26.10.2026.
  0.369.1.50.4795647582.20261021.133)
   case " $compatible " in *" yandex,plum-evt2 "*) return 0;; esac
   ;;
  0.369.1.50.4794654278.20261021.133)
   case " $compatible " in *" yandex,prune-evt2 "*) return 0;; esac
   ;;
  # Station 2: typed VolumeSet 75->55->75 acknowledged 26.10.2026.
  0.369.1.50.4794649087.20261021.295)
   case " $compatible " in *" yandex,station-midi "*) return 0;; esac
   ;;
 esac
 echo 'station-volume: writing is not verified for this model/firmware' >&2
 return 2
}
case "${1:-}" in
 check-write) test $# = 1;check_write;echo ok;;
 get|watch) test $# = 1;exec "$here/bin/native-volume" "$1";;
 set)
  test $# = 2
  check_write
  exec "$here/bin/native-volume" set "$2"
  ;;
 *) echo 'usage: get|watch|check-write|set PERCENT' >&2;exit 2;;
esac
