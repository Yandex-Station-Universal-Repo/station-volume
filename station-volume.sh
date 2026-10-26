#!/bin/sh
# Existing upstream native-volume helper; exact verified model/firmware guards.
# Mini2: native state and typed30->25->30 acknowledgement/restoration verified
#26.10.2026. Other firmware/models are NOT inferred compatible from this test.
set -eu
here=$(dirname "$0")
check_write() {
 build=$(cat /system/vendor/quasar/version_full 2>/dev/null || true)
 case "$build" in
  0.369.1.46.4778262955.20261013.160) return 0;;
  0.369.1.50.4794659041.20261021.299)
   compatible=$(tr '\000' ' ' < /proc/device-tree/compatible)
   case " $compatible " in *" yandex,station-mini2 "*) return 0;; esac
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
