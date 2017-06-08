package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"io/ioutil"
	"io"
	"os"
	"log"
	"path"
	"path/filepath"
	"gopkg.in/telegram-bot-api.v4"
	"encoding/base64"
	"strings"
	"time"
)

var AppPath string

type Config struct {
	Port int		`json:"port"`
	Types []string		`json:"types"`
	LogFile string		`json:"logFile"`
	LogFormat string	`json:"logFormat"`

	BotToken string		`json:"botToken"`
	BotAuth string		`json:"botAuth"`
	BotFormat string	`json:"botFormat"`
}
var config Config

var image []byte

type Admin struct {
	ChatID int64		`json:"chatId"`
}
var admins = []Admin{}

func copyFile(path, dest string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	e := out.Close()
	if e != nil {
		return e
	}

	return nil
}

func init(){
	var err error
	AppPath, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Panic(err)
	}

	if _, err = os.Stat(path.Join(AppPath, "config.json")); os.IsNotExist(err) {
		copyFile(path.Join(AppPath, "resources", "config.json"), path.Join(AppPath, "config.json"))
	}

	if _, err = os.Stat(path.Join(AppPath, "admins.json")); !os.IsNotExist(err) {
		b, err := ioutil.ReadFile(path.Join(AppPath, "admins.json"))
		if err != nil {
			log.Panic(err)
		}
		err = json.Unmarshal(b, &admins)
		if err != nil {
			log.Panic(err)
		}
	}

	b, err := ioutil.ReadFile(path.Join(AppPath, "config.json"))
	if err != nil {
		log.Panic(err)
	}

	err = json.Unmarshal(b, &config)
	if err != nil {
		log.Panic(err)
	}

	image, err = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=")
	if err != nil {
		log.Panic(err)
	}
}

func contains(arr []string, item string) int {
	for k, v := range arr {
		if v == item {
			return k
		}
	}

	return -1
}

func containsChatId(arr []Admin, chatId int64) int {
	for k, v := range arr {
		if v.ChatID == chatId {
			return k
		}
	}

	return -1
}

func listenTelegram(){
	  u := tgbotapi.NewUpdate(0)
	  u.Timeout = 60
	  updates, _ := bot.GetUpdatesChan(u)

	  for update := range updates {
		if update.Message == nil {
			continue
		}

		 if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command(){
				case "auth":
				if containsChatId(admins, update.Message.Chat.ID) > -1 {
					msg.Text = "You are already authorized."
					break
				}

				if config.BotAuth == "" || config.BotAuth != update.Message.CommandArguments() {
					msg.Text = "Sorry, you have provided wrong authentification code."
				}else{
					admins = append(admins, Admin{
						ChatID: update.Message.Chat.ID,
					})
					err := func() error {
						json, err := json.Marshal(admins)
						if err != nil {
							return err
						}
						err = ioutil.WriteFile(path.Join(AppPath, "admins.json"), json, 0666)
						return err
					}()

					if err != nil {
						log.Println("error during saving data", err)
					}

					msg.Text = "You are authrorized as admin."
				}
			}

			bot.Send(msg)
		 }
	  }
}

var bot *tgbotapi.BotAPI

func main(){
	f, err := os.OpenFile(config.LogFile, os.O_CREATE | os.O_APPEND | os.O_WRONLY, 0666)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	if config.BotToken != "" {
		var err error
		bot, err = tgbotapi.NewBotAPI(config.BotToken)

		if err == nil {
			log.Printf("Logged in to telegram bot @%s\n", bot.Self.UserName)

			go listenTelegram()
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		q := r.URL.Query().Get("type")

		if q != "" {
			addr := r.RemoteAddr
			if r.Header.Get("X-Forwarded-For") != "" {
				addr = r.Header.Get("X-Forwarded-For")
			}

			replacer := strings.NewReplacer("{time}", time.Now().Format("2006.01.02 03:04:05"), "{type}", q, "{address}", addr)
			if contains(config.Types, q) > -1 {
				f.WriteString(replacer.Replace(config.LogFormat) + "\n")
			}

			go func(){
				if bot != nil {
					for _, admin := range admins {
						bot.Send(tgbotapi.NewMessage(admin.ChatID, replacer.Replace(config.BotFormat)))
					}
				}
			}()

			fmt.Println(replacer.Replace(config.LogFormat))
			w.Write(image)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}
