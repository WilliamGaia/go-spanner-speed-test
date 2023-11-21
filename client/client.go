package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	UUID          string `json:"uuid"`
	Severity      string `json:"severity,omitempty"`
	StartRequest  string `json:"start_request"`
	GetResponse   string `json:"get_response"`
	ClientElapsed string `json:"client_elapsed"`
}

func callLoopingAPI(url string, interval int) {
	loc, _ := time.LoadLocation("Asia/Taipei")
	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Print(err.Error())
			break
		}
		client_uuid := uuid.New().String()
		req.Header.Set("X-Client-Uuid", client_uuid)

		start_request := time.Now().In(loc)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Print(err.Error())
			break
		}

		defer res.Body.Close()
		get_response := time.Now().In(loc)
		client_elapsed := get_response.Sub(start_request).Microseconds()
		fmt.Printf(get_response.Sub(start_request).String())
		fmt.Printf(fmt.Sprintf("uuid: %s, severity: %s, start_request: %s, get_response: %s, client_elapsed: %s\n",
			client_uuid,
			"WARNING",
			start_request,
			get_response,
			fmt.Sprintf("%v", client_elapsed)))

		// body, err := io.ReadAll(res.Body)
		// if err != nil {
		// 	fmt.Print(err.Error())
		// 	break
		// }
		// fmt.Println(string(body))

		time.Sleep(time.Duration(interval) * time.Second)
	}

}

func main() {
	url := os.Getenv("URL")
	if url == "" {
		url = "http://localhost:8080/startTest"
	}
	callLoopingAPI(url, 1)
}
