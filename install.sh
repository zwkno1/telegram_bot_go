#!/bin/bash

GO_PACKAGES=(
"github.com/go-telegram-bot-api/telegram-bot-api"
"github.com/garyburd/redigo/redis"
"github.com/zwkno1/gojieba"
)

for p in ${GO_PACKAGES[*]}
do
	go get ${p}
done

