package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	maxpages "github.com/max-messenger/gomaxpages"
	maxClinet "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"github.com/max-messenger/maxbot"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	contentDir := filepath.Join(dir, "./demo")

	maxBot(contentDir)
	// or
	//maxClient(contentDir)
}

func maxBot(contentDir string) {
	bot, err := maxbot.NewApi(os.Getenv("BOT_TOKEN"), maxbot.WithHTTPClient(&http.Client{Timeout: 25 * time.Second}))
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
}

func maxClient(contentDir string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api, err := maxClinet.NewApi(os.Getenv("BOT_TOKEN"), maxClinet.WithHTTPClient(&http.Client{Timeout: 25 * time.Second}))
	if err != nil {
		log.Fatal(err)
	}

	pages, err := maxpages.New(contentDir, api.Upload)
	if err != nil {
		log.Fatal(err)
	}

	info, err := api.Bots.GetMyInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("info: %+v", info)

	handle := func(ctx context.Context, update model.Update) {
		fmt.Printf("Received: [%s] %#v\n", update.UpdateType, update)
		switch update.UpdateType {
		case model.UpdateMessageCallback, model.UpdateBotStarted:
			err = pages.HandleApi(ctx, api.Messages, update)
			if err != nil {
				log.Fatal(err)
			}
		default:
			log.Printf("Unknown type: %#v\n", update)
		}
	}

	var updates []model.Update
	var marker int64
	for {
		select {
		case <-ctx.Done():
		default:
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
	}
}
