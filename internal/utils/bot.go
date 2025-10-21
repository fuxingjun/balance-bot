package utils

import (
	"fmt"

	"github.com/fuxingjun/balance-bot/internal/config"
	"github.com/fuxingjun/balance-bot/pkg"
)

func SendWecomMessage(msg string, hook string) error {
	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]any{
			"content": msg,
		},
	}
	_, err := pkg.GetHTTPClient().SendPostRequest(hook, payload, nil, nil)
	return err
}

func SendLarkMessage(msg, hook string) error {
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]any{
			"text": msg,
		},
	}
	_, err := pkg.GetHTTPClient().SendPostRequest(hook, payload, nil, nil)
	return err
}

func SendTelegramMessage(msg, chatId, token string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]any{
		"chat_id":    chatId,
		"text":       msg,
		"parse_mode": "HTML",
	}
	_, err := pkg.GetHTTPClient().SendPostRequest(url, payload, nil, nil)
	return err
}

func SendMessage(msg string) error {
	appConfig, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if appConfig == nil {
		return fmt.Errorf("config is nil")
	}
	if appConfig.Webhook.TelegramToken != "" && appConfig.Webhook.TelegramChatId != "" {
		err := SendTelegramMessage(msg, appConfig.Webhook.TelegramChatId, appConfig.Webhook.TelegramToken)
		if err != nil {
			return err
		}
	}
	if appConfig.Webhook.Wecom != "" {
		err := SendWecomMessage(msg, appConfig.Webhook.Wecom)
		if err != nil {
			return err
		}
	}
	if appConfig.Webhook.Lark != "" {
		err := SendLarkMessage(msg, appConfig.Webhook.Lark)
		if err != nil {
			return err
		}
	}
	return nil
}
