package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type WxPusher struct {
	AppToken string   `json:"app_token,omitempty"`
	TopicIds []int    `json:"topic_ids,omitempty"`
	Uids     []string `json:"uids,omitempty"`
}

func (wx *WxPusher) Send(title string, content string) {
	marshal, err := json.Marshal(map[string]any{
		"appToken":      wx.AppToken,
		"summary":       title,
		"content":       content,
		"contentType":   2,
		"topicIds":      wx.TopicIds,
		"uids":          wx.Uids,
		"verifyPay":     false,
		"verifyPayType": 0,
	})
	if err != nil {
		fmt.Println("marshal WxPusher message error", err)
		return
	}
	req, err := http.NewRequest("POST", "https://wxpusher.zjiecode.com/api/send/message", bytes.NewBuffer(marshal))
	if err != nil {
		fmt.Println("new request error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("send wx_pusher error", err)
		return
	}
	resp.Body.Close()
}
