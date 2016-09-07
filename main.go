package main

import (

	"encoding/json"
	"fmt"
	valid "github.com/asaskevich/govalidator"
	"github.com/boltdb/bolt"
	"github.com/bradialabs/shortid"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strings"


)

var boltDBPath = "/db/url.db"
var shortUrlBkt = []byte("shortUrlBkt")
var dbConn *bolt.DB

type Response struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Url    string `json:"url"`
}

func main() {
	var err error
	dbConn, err = bolt.Open(boltDBPath, 0644, nil)
	if err != nil {
		log.Println(err)
	}

	//defer dbConn.Close()
	router := httprouter.New()
	router.GET("/:code", Redirect)
	router.GET("/:code/json", GetOriginalURL)
	router.POST("/create/", Create)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func Create(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	urlStr := r.FormValue("url")
	urlStr = strings.Trim(urlStr, " ")
	if valid.IsURL(urlStr) == false {
		resp := &Response{Status: http.StatusBadRequest , Msg: "Invalid input URL", Url: ""}
		respJson, _ := json.Marshal(resp)
		fmt.Fprint(w, string(respJson))
		return
	}

	newCode := GetNextCode()
	byteKey, byteUrl := []byte(newCode), []byte(urlStr)
	err := dbConn.Update(func(tx *bolt.Tx) error {
		//@todo : move this code to main function
		bucket, err := tx.CreateBucketIfNotExists(shortUrlBkt)
		if err != nil {
			return err
		}

		err = bucket.Put(byteKey, byteUrl)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Println(err)
		resp := &Response{Status: http.StatusInternalServerError, Msg: "Some error occured while creating short URL:", Url: ""}
		respJson, _ := json.Marshal(resp)
		fmt.Fprint(w, string(respJson))
		return
	}

	shortUrl := newCode
	resp := &Response{Status: http.StatusOK, Msg: "Short URL created successfully", Url: shortUrl}
	respJson, _ := json.Marshal(resp)
	fmt.Fprint(w, string(respJson))
}

func Redirect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	code := ps.ByName("code")
	originalUrl, err := getCodeURL(code)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	http.Redirect(w, r, originalUrl, http.StatusFound)
}

func GetOriginalURL(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	code := ps.ByName("code")
	originalUrl, err := getCodeURL(code)

	if err != nil {
		resp := &Response{Status: http.StatusInternalServerError, Msg: "Some error occured while reading URL", Url: ""}
		respJson, _ := json.Marshal(resp)
		fmt.Fprint(w, string(respJson))
		return
	}

	var resp *Response
	if len(originalUrl) != 0 {
		resp = &Response{Status: http.StatusOK, Msg: "Found", Url: originalUrl}
	} else {
		resp = &Response{Status: http.StatusNotFound, Msg: "URL not found", Url: ""}
	}

	respJson, err := json.Marshal(resp)

	if err != nil {
		fmt.Fprint(w, "Error occurred while creating json response")
		return
	}

	fmt.Fprint(w, string(respJson))
}

func getCodeURL(code string) (string, error) {
	key := []byte(code)
	var originalUrl string

	err := dbConn.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(shortUrlBkt)
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found!", shortUrlBkt)
		}

		value := bucket.Get(key)
		originalUrl = string(value)
		return nil
	})

	if err != nil {
		return "", err
	}
	return originalUrl, nil
}

func GetNextCode() string {
	s := shortid.New()
	return s.Generate()
}

