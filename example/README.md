# Пример бота MAX со страницами

Запускаемый пример бота для мессенджера MAX с многостраничным интерфейсом на базе
[`gomaxpages`](../README.md), `maxbot` и `max-bot-api-client-go`.

Пример показывает два способа подключения библиотеки:

1. **`maxBot`** — высокоуровневый: фреймворк `maxbot` сам ведёт long-polling и вызывает обработчики событий.
2. **`maxClient`** — низкоуровневый: прямая работа с API-клиентом, long-polling и диспетчеризация обновлений вручную.

## Структура

```
example/
├── main.go             # оба режима работы
└── demo/               # контент страниц (уже в репозитории)
    ├── start.md        # стартовая страница (payload = "")
    ├── names.md        # payload = "names"
    ├── attachment.md   # payload = "attachment"
    └── content/        # вложения, загружаются в MAX при старте
        ├── .index      # кэш токенов загрузок
        └── head.jpg
```

Путь к `demo/` в `main.go` вычисляется от расположения самого `main.go` (`runtime.Caller`), поэтому
пример можно запускать из любой директории.

Формат `.md`-страниц описан в [корневом README](../README.md#формат-страницы).

## Требования

- Go 1.25+
- Токен бота MAX — [получение токена](https://dev.max.ru/docs/chatbots/bots-create/manage#%D0%9F%D0%BE%D0%BB%D1%83%D1%87%D0%B5%D0%BD%D0%B8%D0%B5%20%D1%82%D0%BE%D0%BA%D0%B5%D0%BD%D0%B0%20%D0%B1%D0%BE%D1%82%D0%B0)

Пример входит в основной модуль `github.com/max-messenger/gomaxpages`, поэтому зависимости уже
объявлены в `go.mod`. Команды ниже нужны, только если вы копируете пример в собственный проект:

```bash
go get github.com/max-messenger/gomaxpages
go get github.com/max-messenger/max-bot-api-client-go/v2
go get github.com/max-messenger/maxbot
```

## Настройка

```bash
export BOT_TOKEN="ваш_токен_бота"
```

Токены вложений привязаны к загрузившему их боту, поэтому перед запуском со своим `BOT_TOKEN`
сбросьте кэш — иначе `head.jpg` не прикрепится:

```bash
rm -f example/demo/content/.index
```

При следующем старте файлы из `demo/content/` загрузятся заново, а `.index` пересоздастся.

## Выбор режима

В `main()` оставьте нужный вызов:

```go
func main() {
	// ...
	maxBot(contentDir)
	// или
	//maxClient(contentDir)
}
```

## Запуск

```bash
go run ./example        # из корня репозитория
# или
cd example && go run .
```

## Режим `maxBot` (рекомендуемый)

```go
bot, err := maxbot.NewApi(os.Getenv("BOT_TOKEN"),
	maxbot.WithHTTPClient(&http.Client{Timeout: 25 * time.Second}))
if err != nil {
	log.Fatal(err)
}

pages, err := maxpages.New(contentDir, bot.Client().Upload)
if err != nil {
	log.Fatal(err)
}

bot.Handle(maxbot.OnBotStarted, pages.Handle)
bot.Handle(maxbot.OnMessageCallback, pages.Handle)

bot.Start()
```

Обрабатываемые события:

- `OnBotStarted` — пользователь запустил бота;
- `OnMessageCallback` — нажатие inline-кнопки на странице.

## Режим `maxClient`

Ручной long-polling через `api.Subscriptions.GetUpdates` и диспетчеризация обновлений в цикле:

```go
pages, err := maxpages.New(contentDir, api.Upload)
if err != nil {
	log.Fatal(err)
}

for {
	updates, marker, err = api.Subscriptions.GetUpdates(ctx, marker)
	if _, tErr := errors.AsType[*maxClinet.TimeoutError](err); tErr {
		continue
	}
	if err != nil {
		log.Println("GetUpdates: ", err)

		return
	}

	for _, update := range updates {
		handle(ctx, update)
	}
}
```

Внутри `handle` страницы обрабатываются через `pages.HandleApi(ctx, api.Messages, update)`.

Особенности режима:

- `TimeoutError` не прерывает цикл;
- поддерживается отмена через `context.WithCancel`;
- все входящие обновления печатаются в формате `[тип] {данные}`.

## Обрабатываемые типы обновлений

| Тип                           | Описание                                    |
|-------------------------------|---------------------------------------------|
| `model.UpdateBotStarted`      | Пользователь запустил бота                  |
| `model.UpdateMessageCallback` | Нажатие inline-кнопки                       |

Для `UpdateBotStarted` (`payload` пустой) страница `start.md` отправляется новым сообщением,
для `UpdateMessageCallback` — найденная по `payload` страница редактирует текущее сообщение.

## Проверка

В MAX откройте бота и нажмите «Начать»: придёт `start.md` с картинкой и кнопками
«Именование файлов» и «Вложение», которые переключают страницы.

## Полезные ссылки

- [Документация MAX для разработчиков](https://dev.max.ru/)
- [MAX Messenger](https://max.ru/)
