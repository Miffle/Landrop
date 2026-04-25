# Landrop

Небольшой HTTP/WebSocket сервер для передачи файлов и сообщений между устройствами в локальной сети. Не требует интернета, аккаунтов и установки клиента — достаточно открыть браузер.

## Как это работает

Сервер держит WebSocket-соединения со всеми подключёнными клиентами и работает как ретранслятор: файлы разбиваются на чанки по 64 КБ на стороне браузера, каждый чанк упаковывается в JSON и отправляется напрямую нужному получателю по его ID. На диск ничего не пишется, в памяти сервера одновременно находится не более нескольких чанков — подходит для устройств с ограниченным объёмом RAM (тестировалось на Orange Pi).

## Требования

- Go 1.21+
- Устройства должны быть в одной локальной сети

## Установка и запуск

### Локально

```bash
git clone https://github.com/Miffle/Landrop.git
cd landrop
go run ./cmd/server
```

Сервер поднимается на порту `6437` (Можно поменять). Открыть в браузере: `http://localhost:6437`

### На удалённом устройстве (например, Orange Pi)

Сборка под Linux ARM64 на Windows:

```cmd
set GOOS=linux
set GOARCH=arm64
go build -o landrop ./cmd/server
```

Для 32-битных ARM:

```cmd
set GOOS=linux
set GOARCH=arm
set GOARM=7
go build -o landrop ./cmd/server
```

### Копирование на устройство:
Если вы вдруг не делали возможность простого подключения к orangepi (например ssh orangepi), то вместо orangepi:/ - \<user\>@\<ip\>:/
```bash
scp landrop orangepi:/home/orangepi/landrop
scp -r web orangepi:/home/orangepi/landrop
```
Структура, после выполнения команд должна быть такой:
```cmd
landrop.bin  web

./web:
static  templates

./web/static:
app.js  style.css

./web/templates:
index.html
```
### Запуск:

```bash
ssh orangepi
cd /home/orangepi/landrop
chmod +x landrop.bin
./landrop
```

Бинарник должен запускаться из директории, где лежит папка `web/`.

### Автозапуск через systemd

```ini
# /etc/systemd/system/landrop.service
[Unit]
Description=Landrop
After=network.target

[Service]
WorkingDirectory=/home/orangepi/landrop
ExecStart=/home/orangepi/landrop/landrop.bin
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
