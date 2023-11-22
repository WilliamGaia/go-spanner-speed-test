package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

var concurrent int
var loc *time.Location
var url string

type Entry struct {
	UUID          string `json:"uuid"`
	Severity      string `json:"severity,omitempty"`
	StartRequest  string `json:"start_request"`
	GetResponse   string `json:"get_response"`
	ClientElapsed string `json:"client_elapsed"`
}

func callAPI(wg *sync.WaitGroup) {
	defer wg.Done()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Print(err.Error())
	}

	client_uuid := uuid.New().String()
	req.Header.Set("X-Client-Uuid", client_uuid)

	start_request := time.Now().In(loc)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer res.Body.Close()

	get_response := time.Now().In(loc)
	client_elapsed := get_response.Sub(start_request).Microseconds()
	// fmt.Printf(get_response.Sub(start_request).String())
	fmt.Printf(fmt.Sprintf("uuid: %s, severity: %s, start_request: %s, get_response: %s, client_elapsed: %s\n",
		client_uuid,
		"WARNING",
		start_request,
		get_response,
		fmt.Sprintf("%v", client_elapsed)))

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Print(err.Error())
	}
	fmt.Println(string(body))
}

func init() {
	concurrent = 6
	loc, _ = time.LoadLocation("Asia/Taipei")
	url = os.Getenv("URL")
	if url == "" {
		url = "http://localhost:8080/startTest"
	}
}

func main() {
	defer fmt.Println("End Calling API")

	var wg sync.WaitGroup
	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for i := 0; i < concurrent; i++ {
				wg.Add(1)
				go callAPI(&wg)
			}
		}
	}
}
