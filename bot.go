package main

import (
	"encoding/json"
	"flag"
	"github.com/garyburd/redigo/redis"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/zwkno1/gojieba"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMessage string = "Oops!"
)

type BotConfig struct {
	Name    string
	Token   string
	Url     string
	Address string
	Cert    string
	Key     string
}

func SplitText(text string) []string {
	var result []string
	words := jieba.Tag(text)
	allow := []string{"/n", "/nr", "/ns", "/nt", "/nz", "/eng", "/user"}
	for _, word := range words {
		for _, suffix := range allow {
			if strings.HasSuffix(word, suffix) {
				word = word[:len(word)-len(suffix)]
				if len(word) > 1 {
					result = append(result, word)
				}
			}
		}
	}
	log.Println("-------------------------")
	for _, word := range result {
		log.Println(word)
	}
	log.Println("-------------------------")
	return result
}

func GetChatId(message *tgbotapi.Message) string {
	chatId := strconv.FormatInt(message.Chat.ID, 10)
	return message.Chat.Type + chatId
}

func MessageToString(message *tgbotapi.Message) (msg string) {
	jsonMessage, err := json.Marshal(*message)
	if err != nil {
		log.Printf("message marshal error: %+v\n", err.Error())
		msg = message.Text
	} else {
		msg = string(jsonMessage)
	}
	return msg
}

func DefaultMessageHandler(message *tgbotapi.Message) {
	log.Printf("(text) %+v>> \t%+v\n", message.From.FirstName+" "+message.From.LastName, message.Text)
	from := message.From
	if (from != nil) && (message.Chat != nil) {
		userId := strconv.Itoa(from.ID)
		userName := from.FirstName + " " + from.LastName
		chatId := GetChatId(message)
		msg := MessageToString(message)
		words := SplitText(message.Text)
		args := make([]interface{}, len(words)+4)
		args[0] = chatId
		args[1] = userId
		args[2] = msg
		args[3] = userName
		for i, d := range words {
			args[i+4] = d
		}

		reply, err := redis.String(messageHandlerScript.Do(redisClient, args...))
		if err != nil {
			log.Printf("%+v\n", err.Error())
			return
		}
		if reply != "ok" {
			log.Printf("messageHandlerScript reply: %+v\n", reply)
		}
	}
}

func RankHandler(message *tgbotapi.Message) {
	log.Printf("(/rank) %+v>> \t%+v\n", message.From.FirstName+" "+message.From.LastName, message.Text)
	chatId := GetChatId(message)
	reply, err := redis.Strings(rankScript.Do(redisClient, chatId))
	var text string
	if err == nil {
		for i := 0; i < len(reply)-1; i += 2 {
			text += reply[i] + " : " + reply[i+1] + "\n"
		}
	} else {
		log.Print(err)
		text = defaultMessage
	}
	if text == "" {
		text = "empty"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

func TextRankHandler(message *tgbotapi.Message) {
	key := "textrank:" + GetChatId(message)
	reply, err := redis.Strings(redisClient.Do("ZREVRANGE", key, 0, 9, "WITHSCORES"))
	var text string
	if err == nil {
		for i := 0; i < len(reply)-1; i += 2 {
			text += reply[i] + " : " + reply[i+1] + "\n"
		}
	} else {
		log.Println("text rank: ", err)
		text = defaultMessage
	}
	if text == "" {
		text = "empty"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

func TextStudyHandler(message *tgbotapi.Message) {
	var text string

	args := strings.Split(message.CommandArguments(), " ")
	var args2 = make([]interface{}, len(args)+1)
	args2[0] = "textstudy"
	for i, word := range args {
		jieba.AddWord(word, "user")
		args2[i+1] = word
	}

	reply, err := redis.Int64(redisClient.Do("SADD", args2...))
	if err != nil {
		log.Println("text study: ", err)
		text = defaultMessage
	} else {
		text = "study " + strconv.FormatInt(reply, 10) + " words"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

func InfoHandler(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "https://github.com/zwkno1/telegram_bot_go")
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

var bot *tgbotapi.BotAPI
var redisClient redis.Conn
var messageHandlerScript *redis.Script
var rankScript *redis.Script
var jieba *gojieba.Jieba

func loadRedisScript(keyCount int, fileName string) *redis.Script {
	var script *redis.Script
	data, err := ioutil.ReadFile(fileName)
	if err == nil {
		script = redis.NewScript(keyCount, string(data))
	} else {
		log.Print(err)
	}
	return script
}

func loadBotConfig(fileName string) (config BotConfig, err error) {
	var file *os.File
	file, err = os.Open(fileName)
	defer file.Close()
	if err != nil {
		return config, err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}
	log.Println(config.Name)
	return config, err
}

func main() {
	var config BotConfig
	var err error
	cfg := flag.String("c", "./config.json", "config file")
	flag.Parse()
	log.Println("load config file ", *cfg)

	config, err = loadBotConfig(*cfg)
	if err != nil {
		log.Fatal(err)
	}

	//init redis
	redisClient, err = redis.Dial("tcp", "127.0.0.1:6379", redis.DialDatabase(0), redis.DialConnectTimeout(time.Second*3))
	if err != nil {
		log.Fatal(err)
	}

	messageHandlerScript = loadRedisScript(4, "./message_handler.lua")
	rankScript = loadRedisScript(1, "./rank.lua")
	if (rankScript == nil) || (messageHandlerScript == nil) {
		log.Fatal("load redis script failed")
	}

	//init gojieba
	jieba = gojieba.NewJieba()
	key := "textstudy"
	reply, err := redis.Strings(redisClient.Do("SMEMBERS", key))
	if err != nil {
		log.Fatal(err)
	}
	for _, word := range reply {
		jieba.AddWord(word, "user")
	}

	//init bot api
	bot, err = tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	_, err = bot.SetWebhook(tgbotapi.NewWebhook(config.Url + config.Name))
	if err != nil {
		log.Fatal(err)
	}

	updates := bot.ListenForWebhook("/" + config.Name)
	go http.ListenAndServeTLS(config.Address, config.Cert, config.Key, nil)

	messageDispatcher := NewMessageDispatcher(DefaultMessageHandler)
	messageDispatcher.Register("rank", RankHandler)
	messageDispatcher.Register("textrank", TextRankHandler)
	messageDispatcher.Register("textstudy", TextStudyHandler)
	messageDispatcher.Register("info", InfoHandler)

	for update := range updates {
		log.Printf("%+v\n", update)
		if update.Message != nil {
			if update.Message.IsCommand() {
				log.Println("comand: ", update.Message.Command(), "argument: ", update.Message.CommandArguments())
			}
			messageDispatcher.Dispatch(update.Message)
		}
	}
}
