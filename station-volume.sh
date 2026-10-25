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

if [ "${1:-}" = set ]; then
    force=
    for arg in "$@"; do [ "$arg" = --force ] && force=1; done
    build=$(cat /system/vendor/quasar/version_full 2>/dev/null || true)
    if [ -z "$force" ]; then
        case " $verified " in
            *" $build "*) ;;
            *) echo "station-volume: writing is verified only on: $verified (this station: ${build:-unknown}); add --force to try" >&2
               exit 2 ;;
        esac
    fi
    [ $# -ge 2 ] || { echo "usage: set PERCENT [--force]" >&2; exit 2; }
    exec "$here/bin/native-volume" set "$2"
fi
exec "$here/bin/native-volume" "$@"
