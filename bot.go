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
	"regexp"
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
	Redis   string
}

func SplitText(text string) []string {
	var result []string
	words := jieba.Tag(text)
	allow := []string{"/n", "/nr", "/ns", "/nt", "/nz", "/eng", "/user"}
	for _, word := range words {
		for _, suffix := range allow {
			if strings.HasSuffix(word, suffix) {
				word = word[:len(word)-len(suffix)]
				if GetCharNum(word) > 1 {
					result = append(result, word)
				}
			}
		}
	}
	log.Println("[keywords] ", result)
	return result
}

func GetAtUsers(message *tgbotapi.Message) []string {
	reg := regexp.MustCompile(`@[a-zA-Z0-9_]+[a-zA-Z0-9]`)
	ret := reg.FindAll([]byte(message.Text), -1)
	result := make([]string, 0, len(ret))
	for _,s := range ret {
		str := strings.TrimPrefix(string(s), "@")
		if len(str) != 0 {
			result = append(result, str)
		}
	}
	return result
}

func GetCharNum(text string) int {
	return len([]rune(text))
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
	log.Println("(text)")
	from := message.From
	if (from != nil) && (message.Chat != nil) {
		userId := strconv.Itoa(from.ID)
		userName := from.UserName
		nickName := from.FirstName + " " + from.LastName
		chatId := GetChatId(message)
		msg := MessageToString(message)
		words := SplitText(message.Text)
		args := make([]interface{}, len(words)+5)
		args[0] = chatId
		args[1] = userId
		args[2] = msg
		args[3] = userName
		args[4] = nickName
		for i, d := range words {
			args[i+5] = d
		}

		reply, err := redis.String(messageHandlerScript.Do(redisClient, args...))
		if err != nil {
			log.Printf("%+v\n", err.Error())
			return
		}
		if reply != "ok" {
			log.Printf("messageHandlerScript reply: %+v\n", reply)
		}

		// relationship
		replyId := "0"
		if (message.ReplyToMessage != nil) && (message.ReplyToMessage.From != nil) {
			replyId = strconv.Itoa(message.ReplyToMessage.From.ID)
		}
		atUsers := GetAtUsers(message)
		log.Printf("[at users] %+v\n", atUsers)
		args2 := make([]interface{}, len(atUsers)+3)
		args2[0] = chatId
		args2[1] = userId
		args2[2] = replyId
		for i, d := range atUsers {
			args2[i+3] = d
		}

		reply, err = redis.String(relationshipScript.Do(redisClient, args2...))
		if err != nil {
			log.Printf("%+v\n", err.Error())
			return
		}
		if reply != "ok" {
			log.Printf("relationshipScript reply: %+v\n", reply)
		}
	}
}

func RankHandler(message *tgbotapi.Message) {
	log.Println("(rank)")
	chatId := GetChatId(message)
	reply, err := redis.Strings(rankScript.Do(redisClient, "rank", chatId))
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
	log.Println("(textrank)")
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
	log.Println("(textstudy)")
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

func TextBanHandler(message *tgbotapi.Message) {
	log.Println("(textban)")
	var text string

    args := strings.Split(message.CommandArguments(), " ")
    var args2 = make([]interface{}, len(args)+1)
    args2[0] = "banned_words"
    for i, word := range args {
        args2[i+1] = word
    }

    reply, err := redis.Int64(redisClient.Do("SADD", args2...))
    if err != nil {
        log.Println("text ban: ", err)
        text = defaultMessage
    } else {
        text = "ban " + strconv.FormatInt(reply, 10) + " words"
    }

    msg := tgbotapi.NewMessage(message.Chat.ID, text)
    msg.ReplyToMessageID = message.MessageID
    bot.Send(msg)
}

func RelationshipRankHandler(message *tgbotapi.Message) {
	log.Println("(relationship)")
	text := "最想和你搞基的人:\n"
	key := strconv.Itoa(message.From.ID)
	reply, err := redis.Strings(rankScript.Do(redisClient, "relationship", key))
	if err == nil {
		for i := 0; i < len(reply)-1; i += 2 {
			text += reply[i] + " : " + reply[i+1] + "\n"
		}
	} else {
		log.Print(err)
		text = defaultMessage
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

func InfoHandler(message *tgbotapi.Message) {
	log.Println("(info)")
	msg := tgbotapi.NewMessage(message.Chat.ID, "https://github.com/zwkno1/telegram_bot_go")
	msg.ReplyToMessageID = message.MessageID
	bot.Send(msg)
}

var bot *tgbotapi.BotAPI
var redisClient redis.Conn
var messageHandlerScript *redis.Script
var relationshipScript *redis.Script
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

func newRedisClient(address string) redis.Conn {
	client, err := redis.Dial("tcp", address, redis.DialDatabase(0), redis.DialConnectTimeout(time.Second*3))
	if err != nil {
		log.Println("[redis] ", err.Error())
	}
	return client
}

func main() {
	var err error
	var config BotConfig
	cfg := flag.String("c", "./config.json", "config file")
	logPath := flag.String("l", "", "log file")
	flag.Parse()

	//init log 
	var logFile * os.File
	if len(*logPath) != 0 {
		logFile, err = os.OpenFile(*logPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(logFile)
	}
	defer logFile.Close()

	//load config
	log.Println("load config file ", *cfg)
	config, err = loadBotConfig(*cfg)
	if err != nil {
		log.Fatal(err)
	}

	//init redis
	redisClient = newRedisClient(config.Redis)
	if redisClient.Err() != nil {
		log.Fatal("[redis] init failed")
	}

	messageHandlerScript = loadRedisScript(5, "./message_handler.lua")
	relationshipScript = loadRedisScript(3, "./relationship.lua")
	rankScript = loadRedisScript(2, "./rank.lua")
	if (rankScript == nil) || (messageHandlerScript == nil) || (relationshipScript == nil) {
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
	messageDispatcher.Register("textban", TextBanHandler)
	messageDispatcher.Register("gayrank", RelationshipRankHandler)
	messageDispatcher.Register("info", InfoHandler)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		msg := MessageToString(update.Message)
		log.Println("[message] ", msg)

		//check redis connection
		if redisClient.Err() != nil {
			log.Println("[redis] reconnect: ", redisClient.Err())
			redisClient = newRedisClient(config.Redis)
		}

		if redisClient.Err() != nil {
			log.Println("[redis] reconnect failed: ", redisClient.Err())
		}

		messageDispatcher.Dispatch(update.Message)
	}
}
