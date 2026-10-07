# Пример использования MaxBot: simple

- [Пример использования MaxBot: simple](#пример-использования-maxbot-simple)
  - [Обзор](#обзор)
  - [Установка и настройка бота](#установка-и-настройка-бота)
    - [Инициализация бота](#инициализация-бота)
    - [Подключение обработчиков команд](#подключение-обработчиков-команд)
      - [Команда `/help`](#команда-help)
      - [Команда `/command`](#команда-command)
      - [Команда `/reply`](#команда-reply)
    - [Подключение обработчика Callback-запросов](#подключение-обработчика-callback-запросов)
    - [Подключение обработчика событий](#подключение-обработчика-событий)
    - [Подключение обработчика текстовых сообщений](#подключение-обработчика-текстовых-сообщений)
  - [Запуск бота](#запуск-бота)

## Обзор

В файле [`main.go`](main.go) содержится пример с демо-сценариями бота, созданного с помощью фреймворка:

* Ответ на команды (`/help`, `/command`, `/reply`)
* Обработка нажатия на кнопки (callback)
* Подписка на события (например, изменение названия чата)
* Отправка сообщения с клавиатурой
* Логирование входящих текстовых сообщений

## Установка и настройка бота

### Инициализация бота

```go
// Создание списка опций (opts). В примере устанавливается тайм-аут для HTTP-клиента
opts := []maxbot.Opt{
  maxbot.WithHTTPClient(&http.Client{Timeout: 25 * time.Second}),
  // Регистрация подписки на обновления через Webhook
  maxbot.WithWebhook("http://my-bot.cloud.hooli.local/webhook", "secret", []string{
    maxbot.OnBotAdded,
    maxbot.OnMessageCreated,
    maxbot.OnMessageCallback,
  }),
}

// Создание бота с помощью maxbot.NewApi() — используется токен из переменной окружения BOT_TOKEN
token := os.Getenv("BOT_TOKEN")
bot, err := maxbot.NewApi(token, opts...)
if err != nil {
  log.Fatal(err)
}
```

### Подключение обработчиков команд

#### Команда `/help`

```go
// При получении команды `/help` бот создаёт клавиатуру с двумя кнопками: ссылкой и callback-кнопкой
bot.Handle("/help", func (c maxbot.Context) error {
  kb := model.NewKeyboard()
  kb.AddRow().
  // Добавление кнопки-ссылки
  AddLink("Документация", "https://dev.max.ru/docs").
  // Добавление callback-кнопки
  AddCallBack("Нажми на меня", "pushBtn")
  // В ответе бот возвращает сообщение с этой клавиатурой  
  return c.Send("Основная информация:", maxbot.WithKeyboard(kb))
})
```

#### Команда `/command`

Используется для отладки

```go
// При получении команды `/command` бот извлекает структуру команды `(GetCommand())` из контекста
bot.Handle("/command", func (c maxbot.Context) error {
  command := c.Update().GetCommand()
  msg := fmt.Sprintf(
    // Создание структуры команды (`GetCommand()`) 
    "command: %s\nbot name: %s\n params: \n%s\n text: %s\n",
    command.Command, command.BotName,
    strings.Join(command.Params, "\n"),
    command.RemainingText,
  )
  // В ответе бот возвращает извлечённую информацию
  return c.Send(msg)
})
```

#### Команда `/reply`

Используется для ответа на сообщение, которое её вызвало

```go
// При получении команды `/reply` бот создаёт клавиатуру, содержащую ссылку
bot.Handle("/reply", func (c maxbot.Context) error {
  kb := model.NewKeyboard()
  kb.AddRow().
  AddLink("docs", "https://dev.max.ru/docs")
  // В ответе бот использует`c.Reply()` и возвращает клавиатуру, содержащую ссылку
  return c.Reply("reply", maxbot.WithKeyboard(kb))
})
```

### Подключение обработчика Callback-запросов

```go
// Бот связывает callback-данные `pushBtn` с обработчиком
bot.HandleCallback("pushBtn", func (c maxbot.Context) error {
  kb := model.NewKeyboard()
  kb.AddRow().
  AddLink("Документация", "https://dev.max.ru/docs")
  // Когда пользователь нажимает кнопку «Нажми на меня» из команды `/help`, бот отвечает на callback (`c.Answer()`) сообщением «Изменено» и новой клавиатурой
  return c.Answer("Изменено", maxbot.WithKeyboard(kb))
})
```

### Подключение обработчика событий

```go
// Обработчик подписывается на событие `maxbot.OnChatTitleChangedEvent`
bot.Handle(maxbot.OnChatTitleChangedEvent, func (c maxbot.Context) error {
  // При любом изменении названия чата, в котором находится бот, он отправляет уведомление «Заголовок изменён»
  return c.Send("Заголовок изменён")
})
``` 

### Подключение обработчика текстовых сообщений

```go
// Обработчик для `maxbot.OnMessageCreated` подписывется на все текстовые сообщения, которые не являются командами
bot.Handle(maxbot.OnMessageCreated, func (c maxbot.Context) error {
  //err = c.Send(fmt.Sprintf("%s - принято", c.Update().GetMessage().Body.Text))
  //if err != nil {
  //	return err
  //}

  // В этой версии код отправки ответа закомментирован, и бот просто возвращает полученный текст в стандартный вывод (`fmt.Println`). Это использутеся для отладки или логирования
  fmt.Println("-->", c.Update().GetMessage().Body.Text)

  return nil
})
```

## Запуск бота

После подключения обработчиков для запуска бота введите команду:

```go
bot.Start()
```

После запуска бот начинает получать и обрабатывать обновления, используя подключённые обработчики

Вы можете использовать этот код как основу для вашего собственного бота, модифицируя обработчики под ваши задачи

