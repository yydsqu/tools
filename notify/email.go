package notify

import (
	"fmt"
	"gopkg.in/gomail.v2"
)

type Email struct {
	Host      string   `json:"host,omitempty"`
	Port      int      `json:"port,omitempty"`
	Username  string   `json:"username,omitempty"`
	Password  string   `json:"password,omitempty"`
	Recipient []string `json:"recipient,omitempty"`
}

func (e *Email) Send(title string, content string) {
	message := gomail.NewMessage()
	message.SetHeader("From", message.FormatAddress(e.Username, "异常通知"))
	message.SetHeader("To", e.Recipient...)
	message.SetHeader("Subject", title)
	message.SetBody("text/html", content)
	dialer := gomail.NewDialer(
		e.Host,
		e.Port,
		e.Username,
		e.Password,
	)
	if err := dialer.DialAndSend(message); err != nil {
		fmt.Println("send email error:", err)
	}
}
