package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

var wg sync.WaitGroup

var (
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
)

type Event struct {
	Type string `json:"type"`
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
	Payload struct {
		Action  string                   `json:"action"`
		Commits []map[string]interface{} `json:"commits"`
	} `json:"payload"`
}

func eventFormatter(e Event) string {
	switch e.Type {
	case "PushEvent":
		return pushEvent(e)
	case "WatchEvent":
		return watchEvent(e)
	case "IssuesEvent":
		return issuesEvent(e)
	case "ForkEvent":
		return forkEvent(e)
	default:
		return ""
	}
}

func pushEvent(e Event) string {
	if len(e.Payload.Commits) == 0 {
		return ""
	}
	res := fmt.Sprintf("Pushed %s%v commits%s to %s", Blue, len(e.Payload.Commits), Reset, e.Repo.Name)
	return res
}
func watchEvent(e Event) string {
	res := fmt.Sprintf("%sStarred%s %s", Green, Reset, e.Repo.Name)
	return res
}
func issuesEvent(e Event) string {
	actionName := e.Payload.Action
	actionName = fmt.Sprint(strings.ToUpper(actionName[:1]), actionName[1:])

	res := fmt.Sprintf("%s%s%s an issue in %s", Yellow, actionName, Reset, e.Repo.Name)
	return res
}
func forkEvent(e Event) string {
	res := fmt.Sprintf("%sForked%s %s", Green, Reset, e.Repo.Name)
	return res
}

func worker(id int, eventsCh <-chan Event, resultsCh chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for e := range eventsCh {
		message := eventFormatter(e)
		if message != "" {
			resultsCh <- message
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Pls Enter Username")
		return
	}
	userName := os.Args[1]

	url := fmt.Sprintf("https://api.github.com/users/%s/events", userName)

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Network error:", err)
		return
	}

	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		fmt.Println("Invalid User")
		return
	}
	if response.StatusCode != http.StatusOK {
		fmt.Println("Unexpected status:", response.Status)
		return
	}

	// fmt.Printf("type of response body is: %T", response.Body)
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Couldn't read :", err)
		return
	}

	events := []Event{}
	err = json.Unmarshal(data, &events)
	if err != nil {
		fmt.Println("Unmarshal err:", err)
		return
	}

	eventsCh := make(chan Event)
	resultsCh := make(chan string, len(events))
	// fmt.Println("Total events from API:", len(events))

	numWorkers := 4

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(i+1, eventsCh, resultsCh, &wg)
	}
	//Transfering data from struct to channel
	for _, data := range events {
		eventsCh <- data
	}
	close(eventsCh)

	//Closing result channel seperately via goroutine
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	for msg := range resultsCh {
		fmt.Println("-", msg)
	}
}
