<div align="center">
  <img src="assets-src/appicon.png" alt="Proxy Checker Logo" width="128" height="128">
  
  # Proxy Checker
  
  **Десктопное приложение для автоматической проверки прокси-серверов и управления 3proxy**
  
  [![Windows](https://img.shields.io/badge/Windows-10%2F11-blue?logo=windows)](https://www.microsoft.com/windows)
  [![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)
  [![Wails](https://img.shields.io/badge/Wails-v2-red?logo=wails)](https://wails.io/)
  [![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---


> ![Proxy Checker Screenshot](assets-src/screenshot_1.png)

---

**Proxy Checker** — приложение на базе Go + Wails для автоматической проверки списков прокси-серверов и управления 3proxy. Загружает прокси из API или файлов, проверяет их работоспособность и автоматически переключает 3proxy на живые прокси-серверы, а так же автоматически переключает между жими прокси с заданым интервалом.

## Возможности

- 🔄 **Автоматическая проверка прокси** — тестирование HTTP и SOCKS5 прокси на работоспособность
- 📥 **Загрузка из API** — поддержка загрузки прокси из внешних API с авто-обновлением
- 📁 **Импорт из файлов** — загрузка прокси из текстовых файлов и JSON
- 🔁 **Ротация прокси** — автоматическое переключение 3proxy на рабочие прокси-серверы
- 📊 **Мониторинг в реальном времени** — отслеживание статуса прокси и автоматическая смена при падении
- 🛡️ **Встроенный 3proxy** — не требует отдельной установки 3proxy
- 🎯 **Гибкая настройка** — настройка таймаутов, количества потоков, интервалов проверки

## Требования

- **Windows** 10/11
- **Go** 1.21+ (для сборки)
- **Node.js** 18+ (для сборки фронтенда)
- **Wails CLI** v2 (для сборки)
- **Make** (опционально, для упрощения сборки)

## Быстрый старт

> **Для опытных пользователей:**
> ```powershell
> git clone https://github.com/vpnwe912/proxy-checker.git
> cd Proxy-Cheker
> go mod download
> make build  # или: wails build
> .\build\bin\ProxyChecker.exe
> ```

### 1. Установка зависимостей

#### Установка Go

1. Скачайте Go 1.21+ с официального сайта: https://go.dev/dl/
2. Запустите установщик и следуйте инструкциям
3. Проверьте установку:

```powershell
go version
```

Должно вывести: `go version go1.21.x windows/amd64` (или выше)

#### Установка Node.js

1. Скачайте Node.js 18+ (LTS): https://nodejs.org/
2. Запустите установщик (включите опцию "Add to PATH")
3. Проверьте установку:

```powershell
node --version
npm --version
```

#### Установка Wails CLI

После установки Go выполните:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Проверьте установку:

```powershell
wails doctor
```

Эта команда проверит все зависимости и покажет, что установлено корректно.

#### Установка Make (опционально)

Make упрощает команды сборки, но не обязателен.

```powershell
winget install GnuWin32.Make
```

Добавьте Make в PATH (PowerShell от имени администратора):

```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files (x86)\GnuWin32\bin", "Machine")
```

Перезапустите PowerShell и проверьте:

```powershell
make --version
```

### 2. Клонирование проекта

```powershell
git clone <repository-url>
cd Proxy-Cheker
```

### 3. Установка зависимостей проекта

Установите Go модули:

```powershell
go mod download
```

Или используйте `go mod tidy` для очистки и обновления зависимостей:

```powershell
go mod tidy
```

> **Примечание:** 
> - Wails автоматически скачает зависимости при первой сборке, но лучше сделать это явно
> - Встроенный 3proxy уже включен в репозиторий (`embedded\3proxy\`)
> - Зависимости фронтенда (`npm install`) устанавливаются автоматически при сборке через Wails

### 4. Запуск в режиме разработки

**С Make:**

```powershell
make dev
```

**Без Make:**

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\prepare-assets.ps1
wails dev
```

Приложение откроется автоматически. Веб-интерфейс доступен по адресу: `http://127.0.0.1:18080`

### 5. Сборка production-версии

**С Make (рекомендуется):**

```powershell
make build
```

**Без Make:**

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\prepare-assets.ps1
wails build -clean
```

> **Примечание:** Команда `wails build` автоматически выполняет `npm install` и `npm run build` для фронтенда.

Готовый исполняемый файл:

```text
build\bin\ProxyChecker.exe
```

### 6. Первый запуск

1. Запустите приложение:

```powershell
.\build\bin\ProxyChecker.exe
```

2. Откроется окно приложения с веб-интерфейсом на `http://127.0.0.1:18080`

3. При первом запуске автоматически создастся:
   - Папка `runtime\` с конфигурационными файлами
   - Файл `runtime\settings.json` с настройками по умолчанию
   - Встроенный 3proxy распакуется в `runtime\embedded-3proxy\`

4. Настройте приложение через UI:
   - Укажите API URL для загрузки прокси (или загрузите из файла)
   - Настройте параметры проверки
   - Нажмите **"Сохранить настройки"**

5. Готово! Теперь можно:
   - Загружать прокси
   - Проверять их работоспособность
   - Запускать мониторинг и 3proxy

## Команды Make

| Команда | Описание |
|---------|----------|
| `make prepare` | Подготовка иконок приложения |
| `make build` | Полная сборка приложения (prepare + wails build -clean) |
| `make rebuild` | Полная пересборка (clean + build) |
| `make dev` | Запуск в режиме разработки с подготовкой иконок |
| `make clean` | Очистка собранных бинарников (удаление build/bin) |
| `make check` | Проверка наличия необходимых файлов |

## Управление версией

Версия и метаданные приложения хранятся в `wails.json` и автоматически встраиваются в:
- ✅ Исполняемый файл `.exe` (свойства файла Windows)
- ✅ Код приложения (доступна через API `GetVersion()`)

### Автоматическое увеличение версии

При каждом `make build` версия **автоматически увеличивается**:

```
1.0.0 → 1.0.1 → 1.0.2 → ... → 1.0.9 → 1.1.0 → 1.1.1 → ... → 1.1.9 → 1.2.0
```

Просто запустите:

```powershell
make build
```

Версия увеличится автоматически перед сборкой!

### Ручная установка версии

Если нужно установить конкретную версию:

```powershell
.\scripts\set-version.ps1 -Version 2.0.0
```

### Изменение метаданных

Отредактируйте `wails.json`:

```json
{
  "author": {
    "name": "Ваше Имя",
    "email": "your@email.com"
  },
  "info": {
    "companyName": "Ваша Компания",
    "productName": "Proxy Checker",
    "productVersion": "1.0.0",
    "copyright": "Copyright (C) 2026 Ваше Имя",
    "comments": "Описание приложения"
  }
}
```

## Поддерживаемые форматы прокси

### Текстовый формат

```text
127.0.0.1:8085
127.0.0.1:8080@user:pwd
user:pwd@127.0.0.1:8080
127.0.0.1:8080:user:pwd
http://127.0.0.1:8080
socks5://user:pwd@127.0.0.1:1080
```

### JSON формат

Одиночный объект:
```json
{"host":"127.0.0.1","port":"8080","login":"user","password":"pwd"}
```

Массив объектов:
```json
[
  {"host":"127.0.0.1","port":"8085"},
  {"host":"127.0.0.1","port":"8080","login":"user","password":"pwd","type":"socks5"}
]
```

JSON Lines (по одному объекту на строку):
```json
{"host":"127.0.0.1","port":"8085"}
{"host":"127.0.0.1","port":"8080","login":"user","password":"pwd"}
```

## Использование

### Загрузка прокси

1. **Из API:**
   - Укажите URL API в поле "API URL"
   - Нажмите "Загрузить API"
   - Включите "Авто-импорт" для автоматического обновления

2. **Из файла:**
   - Нажмите "Выбрать файл"
   - Выберите файл с прокси (txt, json)

### Проверка прокси

1. Настройте параметры проверки:
   - **URL проверки** — адрес для тестирования (по умолчанию: https://api.ipify.org)
   - **Timeout** — таймаут проверки в секундах
   - **Потоков** — количество параллельных проверок

2. Нажмите **"Проверить все"**

### Мониторинг и ротация

1. Настройте **"Интервал монитора"** (как часто менять прокси)
2. Нажмите **"Старт монитор"**
3. Приложение автоматически:
   - Проверяет прокси
   - Выбирает рабочий прокси
   - Обновляет конфигурацию 3proxy
   - Переключается на следующий прокси через заданный интервал

### Запуск 3proxy

1. Настройте параметры 3proxy:
   - **Proxy port** — порт для входящих подключений (по умолчанию: 8080)
   - **Admin port** — порт администрирования (по умолчанию: 8081)
   - **Allowed IP** — разрешенные IP-адреса (по умолчанию: 127.0.0.1)

2. Нажмите **"Запустить 3proxy"**

3. Используйте прокси:
   ```
   http://127.0.0.1:8080
   ```

## Встроенный 3proxy

Приложение поставляется со встроенным 3proxy. Файлы 3proxy находятся в:

```text
embedded\3proxy\
```

Необходимые файлы:
- `3proxy.exe` — основной исполняемый файл
- `*.dll` — плагины (FilePlugin, PCREPlugin, SSLPlugin, StringsPlugin, TrafficPlugin, WindowsAuthentication, utf8tocp1251)
- `mycrypt.exe` — утилита шифрования

При запуске приложение автоматически извлекает эти файлы в:

```text
runtime\embedded-3proxy\
```

Если поле "Путь к 3proxy.exe" в UI пустое, приложение использует встроенную версию.

## Runtime файлы

Приложение создает следующие файлы во время работы:

```text
runtime\3proxy.cfg      - основная конфигурация 3proxy
runtime\parent.cfg      - конфигурация upstream прокси
runtime\reload.txt      - файл для перезагрузки конфигурации
runtime\logs\3proxy.log - логи 3proxy
```

## Конфигурация

Приложение **автоматически создает** конфигурационный файл `runtime\settings.json` при первом запуске с настройками по умолчанию.

### Настройка через UI

Все параметры настраиваются через графический интерфейс приложения:

1. Запустите приложение
2. Откройте браузер: `http://127.0.0.1:18080`
3. Настройте параметры в левой панели
4. Нажмите **"Сохранить настройки"**

### Параметры по умолчанию

| Параметр | Значение по умолчанию |
|----------|----------------------|
| Адрес веб-интерфейса | `127.0.0.1:18080` |
| URL проверки | `https://api.ipify.org` |
| Timeout проверки | `2` сек |
| Количество потоков | `20` |
| Интервал монитора | `120` сек |
| Тип прокси | `auto` (HTTP и SOCKS5) |
| Proxy port (3proxy) | `8080` |
| Admin port (3proxy) | `8081` |

**⚠️ Важно:** Файлы `settings.json` и `*.cfg` находятся в `.gitignore` и не попадут в репозиторий.

## Структура проекта

```
Proxy-Cheker/
├── assets-src/              # Исходные ресурсы (иконки)
│   └── appicon.png
├── build/                   # Собранные файлы (в .gitignore)
│   ├── bin/
│   │   └── ProxyChecker.exe
│   └── windows/
│       └── icon.ico
├── embedded/                # Встроенные файлы 3proxy
│   └── 3proxy/
├── frontend/                # Фронтенд на Vite + Vanilla JS
│   ├── src/
│   │   ├── main.js
│   │   └── style.css
│   ├── index.html
│   └── package.json
├── scripts/                 # Скрипты сборки
│   └── prepare-assets.ps1
├── wailsjs/                 # Автогенерируемые биндинги Wails
├── desktop_helpers.go       # Вспомогательные функции для десктопа
├── main.go                  # Основная логика приложения
├── proxy.go                 # Парсинг и проверка прокси
├── proxy_test.go            # Unit-тесты
├── threeproxy.go            # Управление 3proxy
├── wails_main.go            # Точка входа Wails
├── Makefile                 # Команды сборки
├── wails.json               # Конфигурация Wails
└── README.md
```

## Тестирование

Unit-тесты **не требуют конфигурации** и запускаются независимо от приложения.

Запуск всех тестов:

```powershell
go test -v
```

Запуск с покрытием кода:

```powershell
go test -v -cover
```

Запуск конкретного теста:

```powershell
go test -v -run TestParseProxyFormats
```

## Разработка

### Архитектура

- **Backend (Go):**
  - `main.go` — HTTP сервер, API endpoints, бизнес-логика
  - `proxy.go` — парсинг прокси из различных форматов, проверка через curl/Go HTTP client
  - `threeproxy.go` — генерация конфигов 3proxy, управление процессом
  - `wails_main.go` — интеграция с Wails, desktop API

- **Frontend (Vanilla JS + Vite):**
  - Минималистичный UI без фреймворков
  - Прямое взаимодействие с Go через Wails bindings
  - Автообновление состояния каждые 2 секунды

### Добавление новых функций

1. Добавьте метод в структуру `DesktopApp` в `wails_main.go`
2. Пересоберите биндинги: `wails dev` автоматически регенерирует `wailsjs/`
3. Используйте новый метод во фронтенде через импорт из `wailsjs/go/main/DesktopApp`

## Устранение проблем

### Приложение не запускается

- Проверьте, что порт 18080 свободен
- Убедитесь, что `settings.json` существует и валиден
- Проверьте логи в `runtime\logs\3proxy.log`

### 3proxy не стартует

- Проверьте, что порты 8080 и 8081 свободны
- Убедитесь, что файлы в `embedded\3proxy\` существуют
- Проверьте права доступа к папке `runtime\`

### Прокси не проверяются

- Убедитесь, что curl установлен (для `useCurl: true`)
- Проверьте доступность `testUrl`
- Увеличьте `checkTimeoutSec` для медленных прокси

## Contributing

Мы приветствуем вклад в проект! Пожалуйста, ознакомьтесь с [CONTRIBUTING.md](CONTRIBUTING.md) для получения информации о:

- Как сообщить об ошибке
- Как предложить новую функцию
- Процесс создания Pull Request
- Стандарты кодирования

## Лицензия

Этот проект распространяется под лицензией MIT. См. файл [LICENSE](LICENSE) для подробностей.

Проект использует следующие компоненты:
- **Wails** — MIT License
- **3proxy** — BSD-like License
- **Go** — BSD-style License

## Code of Conduct

Мы придерживаемся [Code of Conduct](CODE_OF_CONDUCT.md) для создания открытого и дружелюбного сообщества.

## Changelog

Все изменения проекта документируются в [CHANGELOG.md](CHANGELOG.md).

## Контакты

- **GitHub:** [vpnwe912/proxy-checker](https://github.com/vpnwe912/proxy-checker)
- **Issues:** [Сообщить о проблеме](https://github.com/vpnwe912/proxy-checker/issues)
- **Email:** MuPoBoU@ukr.net

Для вопросов и предложений создавайте Issues в репозитории проекта.
