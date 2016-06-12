package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
)

func checkErr(err error, res http.ResponseWriter) bool {
	if err != nil {
		log.Println(err)
		return true
	}
	return false
}

var config map[string]interface{}

func initiate(){
	tmp := make(map[string]interface{})
	tmp["port"] = "4879"
	tmp["mysql-host"] = "localhost"
	tmp["mysql-port"] = "3306"
	tmp["mysql-username"] = "onebone"
	tmp["mysql-password"] = "PASSWORD"
	tmp["mysql-database"] = "logger"
	tmp["types"] = []string{}

	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		content, err := json.MarshalIndent(tmp, "", "\t")
		if err != nil {
			log.Panic(err)
		}

		ioutil.WriteFile("config.json", content, 0666)

		config = tmp
	} else {
		content, _ := ioutil.ReadFile("config.json")
		json.Unmarshal(content, &config)
	}

	for key, val := range tmp {
		if _, ok := config[key]; !ok {
			config[key] = val
		}
	}
}

func connectDatabase() (*sql.DB, error) {
	return sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config["mysql-username"],
		config["mysql-password"], config["mysql-host"],
		config["mysql-port"], config["mysql-database"]))
}

func main(){
	initiate()

	db, err := connectDatabase()
	if err != nil {
		panic(err)
	}

	_, err = db.Query(`CREATE TABLE IF NOT EXISTS addr (
		type VARCHAR(25),
		addr VARCHAR(15),
		time TIME
	)`)
	if err != nil {
		panic(err)
	}

	db.Close()

	log.Fatal(http.ListenAndServe(":" + config["port"].(string), handlers.ProxyHeaders(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		db, err := connectDatabase()
		if checkErr(err, res) {
			return
		}

		stmt, err := db.Prepare("INSERT addr SET type=?, addr=?, time=NOW()")
		if checkErr(err, res) {
			return
		}

		if req.URL.Query().Get("type") != "" {
			for _, t := range config["types"].([]interface{}) {
				if t.(string) == req.URL.Query().Get("type") {
					_, err = stmt.Exec(req.URL.Query().Get("type"), req.RemoteAddr)
					if checkErr(err, res) {
						return
					}
				}
			}

			log.Printf("%s: %s", req.URL.Query().Get("type"), req.RemoteAddr)
		}

		defer db.Close()
	}))))
}
