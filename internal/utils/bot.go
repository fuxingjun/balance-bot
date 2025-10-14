package utils

import (
	"fmt"
)

func SendWecomMessage(msg string, hook string) error {
	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]any{
			"content": msg,
		},
	}
	_, err := GetHTTPClient().SendPostRequest(hook, payload, nil, nil)
	return err
}

func SendLarkMessage(msg, hook string) error {
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]any{
			"text": msg,
		},
	}
	_, err := GetHTTPClient().SendPostRequest(hook, payload, nil, nil)
	return err
}

func SendTelegramMessage(msg, chatId, token string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]any{
		"chat_id":    chatId,
		"text":       msg,
		"parse_mode": "HTML",
	}
	_, err := GetHTTPClient().SendPostRequest(url, payload, nil, nil)
	return err
}
