#!/bin/sh
# Station volume over Quasar IPC (the same volume_set directive as the buttons).
#   yam run station-volume -- get          current percent
#   yam run station-volume -- watch        print every change
#   yam run station-volume -- set 50       set percent (5% steps), waits for Quasar
#   yam run station-volume -- set 50 --force   write on a firmware not listed below
#
# Reading is safe everywhere native-volume finds the ports. Writing is limited to
# firmware builds where the protocol was verified; anything else needs --force.
set -eu
here=$(dirname "$0")
verified="0.369.1.46.4778262955.20261013.160"

check_write() {
    build=$(cat /system/vendor/quasar/version_full 2>/dev/null || true)
    case " $verified " in
        *" $build "*) return 0 ;;
        *) echo "station-volume: writing is not verified on this firmware" >&2; return 2 ;;
    esac
}
if [ "${1:-}" = check-write ]; then
    [ $# -eq 1 ] || exit 2
    check_write
    echo ok
    exit 0
fi
if [ "${1:-}" = set ]; then
    force=
    [ $# -ge 2 ] && [ $# -le 3 ] || { echo "usage: set PERCENT [--force]" >&2; exit 2; }
    if [ $# -eq 3 ]; then [ "$3" = --force ] || exit 2; force=1; fi
    if [ -z "$force" ]; then
        check_write
    fi
    [ $# -ge 2 ] || { echo "usage: set PERCENT [--force]" >&2; exit 2; }
    exec "$here/bin/native-volume" set "$2"
fi
case "${1:-}" in get|watch) [ $# -eq 1 ] || exit 2;; *) echo "usage: get|watch|check-write|set PERCENT [--force]" >&2; exit 2;; esac
exec "$here/bin/native-volume" "$@"
