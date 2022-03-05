package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/olehbozhok/site_block_checker/proxy_util"
	"github.com/olehbozhok/site_block_checker/repo"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getVal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("not found env %s", key)
	}
	return value
}

func main() {
	// loads values from .env into the system
	_ = godotenv.Load(".env")

	db, err := repo.InitDB(
		getVal("DB_USER"),
		getVal("DB_PASS"),
		getVal("DB_HOST"),
		getVal("DB_DBNAME"),
		true,
	)
	checkErr(err)

	// adminUsername := strings.TrimSpace(getVal("TG_ADMIN"))

	bot, err := tgbotapi.NewBotAPI(getVal("TG_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	defer bot.StopReceivingUpdates()

	handleMsg := func(msgUpdate *tgbotapi.Message) {
		cmd := msgUpdate.CommandWithAt()
		switch cmd {
		case "start":
			msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, `Привіт, я створений для моніторингу сайтів зі сторони Білорусі й Росії. Відправляй посилання й я буду перевіряти роботу цього сайту.
Доступні команди для показу активних сайтів
/ru - Росія
/by - Білорусь`)
			go bot.Send(msg)
			return
		case "ru", "by":
			buf := bytes.NewBuffer(nil)
			buf.WriteString("Реєстр сайтів ")

			var list []repo.CheckURLResult
			var err error
			switch cmd {
			case "ru":
				buf.WriteString("Росії")
				list, err = db.GetCheckURLResultMINData("RU")

			case "by":
				buf.WriteString("Білорусі")
				list, err = db.GetCheckURLResultMINData("BY")
			}
			buf.WriteString("\n\n")
			if err != nil {
				log.Println("error on GetCheckURLResult: ", err)
				msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, `Сталась помилка при обробці`)
				msg.ReplyToMessageID = msgUpdate.MessageID
				go bot.Send(msg)
				return
			}

			for _, data := range list {
				s := "❌"
				if data.IsActive {
					s = "✅"
				}
				buf.WriteString(fmt.Sprint(s, data.CheckURL.URL, "\n"))
			}

			msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, buf.String())
			msg.ReplyToMessageID = msgUpdate.MessageID
			msg.DisableWebPagePreview = true
			go bot.Send(msg)
			return

		}

		// handle msg
		msgText := strings.TrimSpace(msgUpdate.Text)
		urlParsed, err := url.Parse(msgText)
		if err != nil {
			msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, `Надішліть, будь ласка, саме посилання на сайт`)
			msg.ReplyToMessageID = msgUpdate.MessageID
			go bot.Send(msg)
			return
		}
		urlParsed.Scheme = "https"

		urlData := repo.CheckURL{URL: urlParsed.String()}
		result, err := checkSite(urlData, &proxy_util.Client{
			Client: resty.NewWithClient(http.DefaultClient),
		}, "")
		if err != nil || !result.IsActive {
			msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, `Дайний сайт й без проксі не доступний =(`)
			msg.ReplyToMessageID = msgUpdate.MessageID
			go bot.Send(msg)
			return
		}

		err = db.AddURL(urlData)
		if err != nil {
			log.Println("error on AddURL:", err)
			msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, `Сталась помилка при додаванні сайту в базу...`)
			msg.ReplyToMessageID = msgUpdate.MessageID
			go bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(msgUpdate.Chat.ID, `Додано, дякую!`)
		msg.ReplyToMessageID = msgUpdate.MessageID
		go bot.Send(msg)
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for update := range updates {

				if update.Message != nil {
					// If we got a message
					handleMsg(update.Message)
				}
			}

		}()
	}

	go startCheck(db)

	wg.Wait()

}
