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
	//"gopkg.in/telegram-bot-api.v4"
	"encoding/base64"
)

var AppPath string

type Config struct {
	Port int		`json:"port"`
	Types []string		`json:"types"`
}
var config Config

var image []byte

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

func main(){
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		q := r.URL.Query().Get("type")
		if q != "" {
			w.Write(image)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil))
}
