# Draft: station-volume `.yampkg`

Упаковка для YAM v2 не меняет `station-volume.sh`, firmware guards, IPC-клиент
или принятые профили. Добавляется только staging recipe, с runtime ID
`<namespace>.volume` внутри YAM-owned immutable payload.

```sh
python3 scripts/build-yampkg.py --namespace station --platform yandexmini_2 \
  --native-volume /outside/build/native-volume-aarch64 \
  --publisher /outside/yam/target/debug/yam-publish --profiles /outside/yam/platforms \
  --work-dir /outside/stage-volume-mini2 --output /outside/volume-mini2.yampkg
```

Python3.11+, host `yam-publish`; три поддержанных targets — Mini2,
yandexplum(Plum/Prune), Station2. Нужен helper AArch64, собранный из этих
исходников. Версия пакета остаётся1.0.2+all4.20261026; права записи разрешаются
только прежним `check-write`/guard внутри shell launcher, не этим форматом.

Весь helper с LICENSE живёт под `share/luna/plugins/<namespace>.volume/`.
Общий bin/lib не меняется, rootfs/инициализация/CA не затрагиваются.
Скрипт вызывает `yam-publish pack`, не публикует и не устанавливает.
Namespace `station` в примере не зарегистрирован этим PR.

Для использования новым AirPlay сначала нужны Luna #5/#6 и AirPlay #12/#13;
provider нужно включить в Luna как utility, не как постоянный service.
Отдельного supervisor в YAM v2 эта упаковка не добавляет. Черновик до
согласования схемы и повторной аппаратной приёмки.
