package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"fmt"
	"io/ioutil"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"strings"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/golang/freetype"
	"golang.org/x/image/math/fixed"
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
	tmp["header-message"] = "조회 트래커\n당신의 아이피: %s"
	tmp["types"] = make(map[string][]string)

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

func writeImage(req *http.Request, res http.ResponseWriter, message string){
	dst := image.NewRGBA(image.Rect(0, 0, 400, 100))
	draw.Draw(dst, dst.Bounds(), image.White, image.ZP, draw.Src)

	b, _ := ioutil.ReadFile("font.ttf")
	font, err := freetype.ParseFont(b)
	checkErr(err, res)

	c := freetype.NewContext()
	c.SetDst(dst)
	c.SetClip(dst.Bounds())
	c.SetSrc(image.Black)
	c.SetFont(font)
	c.SetFontSize(15)

	point, err := drawText(fmt.Sprintf(config["header-message"].(string), req.RemoteAddr), c, freetype.Pt(60, 20))
	checkErr(err, res)
	_, err = drawText(message, c, freetype.Pt(60, point.Y.Round() + 30))
	checkErr(err, res)

	var opt jpeg.Options
	opt.Quality = 80

	buf := bytes.NewBuffer([]byte{})
	jpeg.Encode(buf, dst, &opt)
	res.Write(buf.Bytes())
}

func drawText(msg string, c *freetype.Context, point fixed.Point26_6) (fixed.Point26_6, error) {
	for _, m := range strings.Split(msg, "\n") {
		p, err := c.DrawString(m, point)
		point = freetype.Pt(60, p.Y.Round() + 20)

		if err != nil {
			return point, err
		}
	}
	return point, nil
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
		defer db.Close()
		if checkErr(err, res) {
			return
		}

		stmt, err := db.Prepare("INSERT addr SET type=?, addr=?, time=NOW()")
		if checkErr(err, res) {
			return
		}

		if req.URL.Query().Get("type") != "" {
			log.Printf("%s: %s", req.URL.Query().Get("type"), req.RemoteAddr)

			for t, msg := range config["types"].(map[string]interface{}) {
				if t == req.URL.Query().Get("type") {
					msgs := msg.([]interface{})
					var m string
					if len(msgs) > 1 {
						m = msgs[rand.Intn(len(msgs) - 1)].(string)
					}else{
						m = msgs[0].(string)
					}

					writeImage(req, res, m)
					_, err = stmt.Exec(req.URL.Query().Get("type"), req.RemoteAddr)
					checkErr(err, res)
					return
				}
			}
		}

		writeImage(req, res, "")
	}))))
}
