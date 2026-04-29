# Landrop

Небольшой HTTP/WebSocket сервер для передачи файлов и сообщений между устройствами в локальной сети. Не требует интернета, аккаунтов и установки клиента — достаточно открыть браузер.

## Как это работает

Сервер держит WebSocket-соединения со всеми подключёнными клиентами и работает как ретранслятор: файлы разбиваются на чанки по 64 КБ на стороне браузера, каждый чанк упаковывается в JSON и отправляется напрямую нужному получателю по его ID. На диск ничего не пишется, в памяти сервера одновременно находится не более нескольких чанков — подходит для устройств с ограниченным объёмом RAM (тестировалось на Orange Pi).

## Требования

- Go 1.21+
- Устройства должны быть в одной локальной сети

## Установка и запуск

### Локально

1) Заходим в релизы и скачиваем последнюю версию
2) Запускаем

Сервер поднимается на порту `6437` (Можно поменять). Открыть в браузере: `http://localhost:6437`

### На удалённом устройстве (например, Orange Pi)

Достаточно будет выполнить команду:
```shell
sh <(wget -O - https://raw.githubusercontent.com/Miffle/Landrop/main/install.sh)
```
Или
```shell
wget -O /tmp/install.sh https://raw.githubusercontent.com/Miffle/Landrop/main/install.sh && sh /tmp/install.sh
```
> [!IMPORTANT]
> В пункте "[landrop] Install as systemd service? [Y/n]" советую выбрать Y 
> 
### Запуск:
1) Переходим в каталог landrop 
```bash
cd landrop
```
2) Запускаем файл
```shell
./landrop
```

### Автозапуск через systemd

```ini
# /etc/systemd/system/landrop.service
[Unit]
Description=Landrop
After=network.target

[Service]
WorkingDirectory=/home/orangepi/landrop
ExecStart=/home/orangepi/landrop/landrop
Restart=on-failure
User=orangepi
```
## Структура проекта
```
cmd/server/        — точка входа 
internal/
  presence/        — хаб: реестр клиентов, маршрутизация сообщений 
  protocol/        — типы сообщений (JSON)
  server/          — WebSocket handler
web/
  templates/       — index.html
  static/          — app.js, style.css
```

## Ограничения

- Нет авторизации — не запускайте в публичных сетях
- Передача только между двумя конкретными клиентами за раз (one-to-one)

[![Star History Chart](https://api.star-history.com/chart?repos=miffle/landrop&type=date&legend=top-left)](https://www.star-history.com/?repos=miffle%2Flandrop&type=date&legend=top-left)
