package notify

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	Token  string `json:"token"`
	ChatId int64  `json:"chat_id"`
}

func (t *Telegram) Send(title string, content string) {
	bot, err := tgbotapi.NewBotAPI(t.Token)
	if err != nil {
		fmt.Println("new bot error:", err)
		return
	}
	msg := tgbotapi.NewMessage(t.ChatId, fmt.Sprintf("<b>%s</b> (%s)\n", title, content))
	msg.ParseMode = "HTML"
	msg.DisableWebPagePreview = true
	if _, err = bot.Send(msg); err != nil {
		fmt.Println("send telegram message error", err)
	}
}
