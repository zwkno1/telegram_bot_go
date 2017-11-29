FROM base/archlinux
MAINTAINER zwkno1@gmail.com

RUN pacman -Syu --noconfirm
RUN pacman -S git gcc go --noconfirm

RUN go get github.com/go-telegram-bot-api/telegram-bot-api
RUN go get github.com/garyburd/redigo/redis
RUN go get github.com/zwkno1/gojieba

RUN mkdir -p /usr/local/bot
WORKDIR /usr/local/bot

ADD bot.go /usr/local/bot  
ADD message_dispatcher.go /usr/local/bot  
ADD message_handler.lua /usr/local/bot  
ADD rank.lua /usr/local/bot  
ADD relationship.lua /usr/local/bot

RUN cd /usr/local/bot && go build -o bot bot.go message_dispatcher.go 

ENTRYPOINT [ "/usr/local/bot/bot", "-c", "/data/config.json", "-l", "/data/bot.log" ]

