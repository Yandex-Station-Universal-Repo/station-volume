# station-volume

YAM-пакет для Яндекс Станции Midi: громкость станции через Quasar IPC, той же директивой `volume_set`, что шлют кнопки. Quasar принимает уровень, кольцо его показывает, приложение синхронизируется. `amixer set Master` так не умеет: на Midi это регулятор только звука Алисы, и Quasar его перезаписывает ([nyakokitsu/yam#5](https://github.com/nyakokitsu/yam/issues/5)).

```sh
yam install station-volume
yam enable station-volume
yam run station-volume -- get            # текущий процент
yam run station-volume -- set 45         # шаг 5%, ждёт подтверждения от Quasar
yam run station-volume -- watch          # изменения пишутся в yam logs, до yam stop
```

Чтение работает везде, где в `/system/vendor/quasar/quasar.cfg` есть порты `stereo_pair` и `iohub_clients`. Запись разрешена только на проверенной сборке `0.369.1.46.4778262955.20261013.160`; на другой нужен `set N --force`.

Протокол. Состояние читается из `stereo_pair`: сообщение 198, поле `state.11` (`6` — процент float32, `3` — mute). Команда идёт в `iohub_clients`: `IOControl.external_directives.directive_list` → `TDirective{name="volume_set", DirectiveType=VolumeSet(254), PercentLevel, Source=Button}`. Поля взяты из [v3rt3p/quasar-protos](https://github.com/v3rt3p/quasar-protos); обёртка (поле 252 с телом, 253 с типом) выведена из наблюдения за потоками Midi. Учётные данные, микшер и другие сервисы не трогаются.

Сборка: `./build.sh` (нужен Go) → `station-volume-<version>.yap`.
