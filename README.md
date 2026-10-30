# gomaxpages

## Описание

Это Golang-библиотека для ботов MAX, которая превращает набор MD-файлов в многостраничный интерфейс: страницы с inline-кнопками, навигацию по callback-ам и автоматическую загрузку вложений

Особенности библиотеки:

- Декларативные страницы — интерфейс описывается в MD-файлах, а не в коде
- Навигация по callback-ам — кнопки переключают страницы и редактируют текущее сообщение
- Автоматическая загрузка вложений — файлы из `content/` загружаются в MAX при старте и кешируются
- Совместимость — работает как с официальной Golang-библиотекой [`MAX Bot API`](https://github.com/max-messenger/max-bot-api-client-go), так и с Golang-фреймворком [`maxbot`](https://github.com/max-messenger/maxbot)

## Системные требования

Go 1.25+

## Установка библиотеки

Для установки используйте команду `go get`:

```bash
go get github.com/max-messenger/gomaxpages
```

## Подготовка к запуску

Для запуска бота необходимо создать тестовые страницы

### Шаблон страницы

Страницы используют формат шаблона из Go-пакета [`max-message-template-go`](/docs/chatbots/bots-coding/go#Go%20template). На страницы можно добавить вложения

```
image:img.jpg
file:doc.pdf

===

Текст страницы с подстановкой {заполнитель}
---
[кнопка1](вызов1), [кнопка2](вызов2)
[кнопка3](вызов3)
```

Синтаксис шаблона:

- `===` — разделитель блока вложений от тела страницы
- `---` — разделитель тела страницы и клавиатуры
- `[имя](вызов)` — кнопка, в одной строке несколько кнопок разделяются запятой
- `{заполнитель}` — место для подстановки значения из вложений

### Типы кнопок

| Тип | Префикс | Пример |
| --- | --- | --- |
| Callback | `cb:` | `[Голос](cb:vote_yes)` |
| Внешняя ссылка | `http(s)://` | `[Сайт](https://max.ru)` |
| Геолокация | `geo:true\|false` | `[Местоположение](geo:true)` |
| Отправка контакта | `contact:` | `[Поделиться контактом](contact:)` |
| Открытие мини-приложения | `app:<bot_id>` | `[Приложение](app:123456)` |
| Отправка сообщения | `msg:` | `[Ответить](msg:привет)` |
| Копирование в буфер | `clip:` | `[Копировать](clip:secret_code)` |

### Вложения

Все файлы из `content/` загружаются в MAX при вызове конструктора `New` (таймаут — 5 минут, до 3 попыток на файл)

Тип загрузки выбирается по расширению:

- `.jpg` / `.jpeg` / `.png` — изображение
- `.mp4` — видео
- `.mp3` — аудио
- Остальные — файл

Полученные токены кешируются в `content/.index` в формате `относительный/путь:токен`, поэтому при повторном запуске файлы не загружаются заново. Если изменили файл, удалите его строку из `.index` (или удалите весь файл)

В шаблоне вложение подключается одним из способов:

- Директивой с относительным путём от `content/`: `image:`, `video:`, `audio:`, `file:`, `sticker:`
- Прямой ссылкой `http(s)://`

### Структура файлов

```
demo/
├── start.md            # Стартовая страница (payload = "")
├── help.md             # payload = "help"
├── legal/
│   └── privacy.md      # payload = "legal/privacy"
└── content/            # Вложения, загружаются в MAX при вызове конструктора New
    ├── .index          # Кеш токенов (создаётся автоматически)
    └── head.jpg        # Доступен в шаблонах как head.jpg
```

Имя файла без расширения `.md` совпадает с `payload` кнопки; вложенные директории задаются через `/`.

### Тестовые данные

В качестве примера создадим только две страницы – `start.md` и `help.md`, где:

- `start.md` — стартовая страница, которая открывается при старте бота (пустой `payload`). В ней отображается картинка `head.jpg`, приветствие `{user}` и кнопки навигации
- `help.md` — вторая страница, на которую ведёт кнопка **Вперёд**. В неё подставляется список доступных команд из `{commands}` — заполнитель, значение которого задаётся в `pages.Values`, см. [Пример быстрого старта](#пример-быстрого-старта)

Файл `demo/start.md`:

```
image:head.jpg

===

Добро пожаловать, {user}!
---
[Вперёд](cb:help)
[Сайт](https://max.ru)
```

Файл `demo/help.md`:

```
Доступные команды:

{commands}
---
[Назад](cb:start)
```

### Безопасность

- `payload` проверяется регулярным выражением `^[a-zA-Z0-9_\-/]+$`;
- Сегменты `..` и пустые сегменты отбрасываются;
- Итоговый путь проверяется на принадлежность `contentDir`

Некорректный `payload` приводит к пустому ответу (страница не показывается), отсутствующий файл — к `ErrNotFound`

## API

| Сигнатура | Назначение |
| --- | --- |
| `New(contentDir string, uploader UploadAPI) (*Pages, error)` | Загружает вложения из `contentDir/content/` и создаёт `Pages` |
| `(*Pages).Handle(ctx maxbot.Context) error` | Обработчик для `maxbot` |
| `(*Pages).HandleApi(ctx context.Context, api API, update model.Update) error` | Обработчик для «чистого» API-клиента |

Логика обработчиков одинакова:

- Пустой `payload` (старт бота) — страница `start.md` отправляется новым сообщением
- Любой другой `payload` (нажатие кнопки) — соответствующая страница редактирует текущее сообщение

Ошибки:

- `ErrContextDirEmpty` — пустой `contentDir`
- `ErrTraversalAttempt` — выход за пределы `contentDir`
- `ErrNotFound` — страница отсутствует

## Пример быстрого старта

1. [Создайте бота](https://dev.max.ru/docs/chatbots/bots-create/create) на платформе MAX для партнёров
2. [Получите токен бота (access_token)](https://dev.max.ru/docs/chatbots/bots-create/manage#%D0%9F%D0%BE%D0%BB%D1%83%D1%87%D0%B5%D0%BD%D0%B8%D0%B5%20%D1%82%D0%BE%D0%BA%D0%B5%D0%BD%D0%B0%20%D0%B1%D0%BE%D1%82%D0%B0) и установите его как переменную окружения `BOT_TOKEN`
3. Подготовьте директорию со страницами:

    ```
    demo/
    ├── start.md            # Стартовая страница (payload = "")
    ├── help.md             # payload = "help"
    ```

4. Создайте файл `main.go` и вставьте следующий код:

    ```go
    package main

    import (
        "log"
        "os"

        "github.com/max-messenger/max-message-template-go"
        "github.com/max-messenger/gomaxpages"
        "github.com/max-messenger/maxbot"
    )

    func main() {
        // Получение токена бота из переменных окружения
        access_token := os.Getenv("BOT_TOKEN")

        // Создание нового экземпляра бота
        bot, err := maxbot.NewApi(access_token)
        if err != nil {
            log.Fatal(err)
        }

        // Загрузка страниц и вложений
        pages, err := maxpages.New("./demo", bot.Client().Upload)
        if err != nil {
            log.Fatal(err)
        }

        // Заполнение значений
        pages.Values = template.Values{
            "user": "Алиса",
            "commands": "/start — открыть бота\n/help — справка",
        }

        // Обработка старта бота и нажатий кнопок
        bot.Handle(maxbot.OnBotStarted, pages.Handle)
        bot.Handle(maxbot.OnMessageCallback, pages.Handle)

        // Запуск бота и начало мониторинга событий
        log.Println("Бот запускается...")
        bot.Start()
    }
    ```
    
    Без `maxbot` — через `HandleApi` с любым типом, реализующим `Send` и `EditMessage`, например в `api.Messages`:

    ```go
    api, err := maxClient.NewApi(os.Getenv("BOT_TOKEN"))
    if err != nil {
        log.Fatal(err)
    }

    pages, err := maxpages.New("./demo", api.Upload)
    if err != nil {
        log.Fatal(err)
    }

    switch update.UpdateType {
    case model.UpdateBotStarted, model.UpdateMessageCallback:
        err = pages.HandleApi(ctx, api.Messages, update)
    }
    ```

    Полный пример режимов с `maxbot` и без — в [`example/`](./example)

5. Для запуска бота в терминале выполните команду:

    ```bash
    go run main.go
    ```

    После выполнения команды в терминале отобразится сообщение: `Бот запускается...`

6. Откройте диалог с вашим ботом в MAX и нажмите «Начать» — это соответствует событию `bot_started` с пустым `payload`. Бот отправит страницу `start.md`, изображение `head.jpg`, приветствие с подставленным именем и клавиатуру:

    ```
    <head.jpg>

    Добро пожаловать, Алиса!
    ---
    [Вперёд](cb:help)
    [Сайт](https://max.ru)
    ```

    После нажатия **Вперёд** бот отправит страницу `help.md`:

    ```
    Доступные команды:

    /start — открыть бота
    /help — справка
    ---
    [Назад](cb:start)
    ```

Готово! Ваш бот работает