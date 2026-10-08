# gomaxpages

Библиотека для ботов MAX Messenger, которая превращает набор `.md`-файлов в многостраничный интерфейс:
страницы с inline-кнопками, навигация по callback-ам и автоматическая загрузка вложений.

Работает как с фреймворком [`maxbot`](https://github.com/max-messenger/maxbot), так и напрямую с
API-клиентом [`max-bot-api-client-go`](https://github.com/max-messenger/max-bot-api-client-go).

## Установка

```bash
go get github.com/max-messenger/gomaxpages
```

Требования: Go 1.25+.

## Быстрый старт

```go
bot, err := maxbot.NewApi(os.Getenv("BOT_TOKEN"))
if err != nil {
	log.Fatal(err)
}

pages, err := maxpages.New("./demo", bot.Client().Upload)
if err != nil {
	log.Fatal(err)
}

bot.Handle(maxbot.OnBotStarted, pages.Handle)
bot.Handle(maxbot.OnMessageCallback, pages.Handle)

bot.Start()
```

Без `maxbot` — через `HandleApi` с любым типом, реализующим `Send` и `EditMessage` (например, `api.Messages`):

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

Полный пример обоих режимов — в [`example/`](./example).

## API

| Сигнатура | Назначение |
|---|---|
| `New(contentDir string, uploader UploadAPI) (*Pages, error)` | Загружает вложения из `contentDir/content/` и создаёт `Pages` |
| `(*Pages).Handle(ctx maxbot.Context) error` | Обработчик для `maxbot` |
| `(*Pages).HandleApi(ctx context.Context, api API, update model.Update) error` | Обработчик для «чистого» API-клиента |

Логика обработчиков одинакова: пустой `payload` (старт бота) — страница `start.md` отправляется новым
сообщением, любой другой `payload` (нажатие кнопки) — соответствующая страница редактирует текущее сообщение.

Ошибки: `ErrContextDirEmpty` (пустой `contentDir`), `ErrTraversalAttempt` (выход за пределы `contentDir`),
`ErrNotFound` (страница отсутствует).

## Структура директории

```
demo/
├── start.md            # стартовая страница (payload = "")
├── names.md            # payload = "names"
├── legal/
│   └── privacy.md      # payload = "legal/privacy"
└── content/            # вложения, загружаются в MAX при New()
    ├── .index          # кэш токенов (создаётся автоматически)
    └── head.jpg        # доступен в шаблонах как head.jpg
```

Имя файла без расширения `.md` совпадает с `payload` кнопки; вложенные директории задаются через `/`.

## Формат страницы

Файлы используют формат [`max-message-template-go`](https://github.com/max-messenger/max-message-template-go):

```
image:head.jpg
file:price.pdf

===

Текст страницы с подстановкой {ключ}
---
[Вперёд](cb:names), [Назад](cb:start)
[Сайт](https://max.ru)
```

- `===` — отделяет блок вложений от тела страницы;
- `---` — отделяет тело от клавиатуры;
- `[имя](вызов)` — кнопка, в одной строке несколько кнопок разделяются запятой;
- `{ключ}` — подстановка значения из вложений.

Типы кнопок: `cb:`, `http(s)://`, `geo:`, `contact:`, `app:`, `msg:`, `clip:`.

## Вложения

Все файлы из `content/` загружаются в MAX при вызове `New` (таймаут 5 минут, до 3 попыток на файл).
Тип загрузки выбирается по расширению: `.jpg`/`.jpeg`/`.png` — изображение, `.mp4` — видео,
`.mp3` — аудио, остальные — файл.

Полученные токены кэшируются в `content/.index` в формате `относительный/путь:токен`, поэтому при
повторном запуске файлы не загружаются заново. Изменили файл — удалите его строку из `.index`
(или весь файл).

В шаблоне вложение подключается директивой с относительным путём от `content/`:
`image:`, `video:`, `audio:`, `file:`, `sticker:`, либо прямой ссылкой `http(s)://`.

## Безопасность

- `payload` проверяется регулярным выражением `^[a-zA-Z0-9_\-/]+$`;
- сегменты `..` и пустые сегменты отбрасываются;
- итоговый путь проверяется на принадлежность `contentDir`.

Некорректный `payload` приводит к пустому ответу (страница не показывается), отсутствующий файл — к `ErrNotFound`.
