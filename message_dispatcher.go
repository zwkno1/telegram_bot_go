package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"strings"
)

type HandlerFunc func(*tgbotapi.Message)

type MessageDispatcher struct {
	handlers       map[string]HandlerFunc
	defaultHandler HandlerFunc
}

func NewMessageDispatcher(defaultHandler HandlerFunc) *MessageDispatcher {
	result := new(MessageDispatcher)
	result.handlers = make(map[string]HandlerFunc)
	result.defaultHandler = defaultHandler
	return result
}

func (d *MessageDispatcher) Register(command string, handler HandlerFunc) {
	if strings.HasPrefix(command, "/") {
		command = strings.TrimPrefix(command, "/")
	}
	d.handlers[command] = handler
}

func (d *MessageDispatcher) Dispatch(message *tgbotapi.Message) {
	handler := d.defaultHandler
	if message.IsCommand() {
		h, ok := d.handlers[message.Command()]
		if ok {
			handler = h
		}
	}

	if handler != nil {
		handler(message)
	}
}
