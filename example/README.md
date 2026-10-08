# Max Messenger Bot с поддержкой страниц

Пример бота для мессенджера MAX с поддержкой многостраничных интерфейсов (страниц), реализованного на Go с использованием библиотек
`maxbot`, `max-bot-api-client-go` и `gomaxpages`.

## Описание

Проект демонстрирует два подхода к созданию бота:

1. **`maxBot`** — высокоуровневый подход с использованием фреймворка `maxbot` (обработчики событий, встроенный цикл получения обновлений).
2. **`maxClient`** — низкоуровневый подход с прямым использованием API-клиента `max-bot-api-client-go` (ручное управление long-polling и
   обработкой обновлений).

Оба подхода используют библиотеку `gomaxpages` для организации многостраничных сценариев (страницы с кнопками, навигация, callback-и).

## Структура проекта

```
.
├── main.go          # основной файл с примерами
└── demo/            # директория с контентом страниц (создаётся вручную)
    └── ...          # файлы описания страниц
```

> Директория `./demo` должна содержать контент страниц, используемый библиотекой `gomaxpages`. Путь вычисляется относительно расположения
`main.go`.

## Требования

- Go 1.21+ (или версия, требуемая зависимостями)
- Токен бота MAX, полученный у [@MasterBot](https://max.ru/)

## Зависимости

```bash
go get github.com/max-messenger/gomaxpages
go get github.com/max-messenger/max-bot-api-client-go/v2
go get github.com/max-messenger/maxbot
```

## Настройка

Установите переменную окружения с токеном бота:

```bash
export BOT_TOKEN="ваш_токен_бота"
```

## Запуск

В функции `main` выберите нужный режим — раскомментируйте один из вызовов:

```go
func main() {
// ...
maxBot(contentDir)
// или
//maxClient(contentDir)
}
```

Затем выполните:

```bash
go run main.go
```

## Режимы работы

### Режим `maxBot` (рекомендуемый)

Использует фреймворк `maxbot` для декларативной обработки событий:

```go
bot, err := maxbot.NewApi(os.Getenv("BOT_TOKEN"),
maxbot.WithHTTPClient(&http.Client{Timeout: 25 * time.Second}))

pages, err := maxpages.New(contentDir, bot.Client().Upload)

bot.Handle(maxbot.OnBotStarted, pages.Handle)
bot.Handle(maxbot.OnMessageCallback, pages.Handle)

bot.Start()
```

Обрабатываются события:

- `OnBotStarted` — запуск бота пользователем (обычно `/start`).
- `OnMessageCallback` — нажатие inline-кнопок на страницах.

### Режим `maxClient`

Ручной long-polling через `api.Subscriptions.GetUpdates` с обработкой обновлений в цикле:

```go
for {
updates, marker, err = api.Subscriptions.GetUpdates(ctx, marker)
// ...
for _, update := range updates {
handle(ctx, update)
}
}
```

Особенности:

- Корректно обрабатывается `TimeoutError` (продолжает цикл без выхода).
- Поддерживается отмена через `context.WithCancel`.
- Выводятся все входящие обновления в формате `[тип] {данные}`.

## Обрабатываемые типы обновлений

| Тип                           | Описание                   |
|-------------------------------|----------------------------|
| `model.UpdateBotStarted`      | Пользователь запустил бота |
| `model.UpdateMessageCallback` | Нажатие inline-кнопки      |

## Лицензия

См. лицензии используемых библиотек:

- [max-messenger/gomaxpages](https://github.com/max-messenger/gomaxpages)
- [max-messenger/max-bot-api-client-go](https://github.com/max-messenger/max-bot-api-client-go)
- [max-messenger/maxbot](https://github.com/max-messenger/maxbot)

## Полезные ссылки

- [Документация MAX для разработчиков](https://dev.max.ru/)
- [MAX Messenger](https://max.ru/)