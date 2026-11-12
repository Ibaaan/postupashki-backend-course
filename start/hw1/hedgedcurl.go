package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

var timeout = time.Duration(15 * int(time.Second))

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		os.Exit(0)
	}

	switch args[0] {
	case "-h":
		printHelp()
	case "--help":
		printHelp()
	case "-t":
		args = setTimeout(args)
	case "--timeout":
		args = setTimeout(args)
	}

	done := make(chan Resp, len(args))
	client := http.Client{}
	ctx, cancel := context.WithCancel(context.Background())
	for _, v := range args {
		go getURLWithTimeout(ctx, v, done, &client)
	}

	hadTimeout := false
	succeded := false

	for i := 0; i < len(args); i++ {
		response := <-done
		if response.r != nil && response.r.StatusCode >= 200 && response.r.StatusCode < 300 {
			succeded = true
			fmt.Printf("%s %s\nheaders: %v\nbody:%s\n",
				response.r.Proto,
				response.r.Status,
				response.r.Header,
				string(response.body),
			)
			break
		}
		if response.err != nil && errors.Is(response.err, context.DeadlineExceeded) {
			hadTimeout = true
		}
	}
	cancel()

	if hadTimeout {
		os.Exit(228)
	}
	if !succeded {
		os.Exit(1)
	}
}

func setTimeout(args []string) []string {
	num, _ := strconv.Atoi(args[1])
	timeout = time.Duration(num * int(time.Second))
	return args[2:]
}

func printHelp() {
	fmt.Println("Help")
	os.Exit(0)
}

type Resp struct {
	r    *http.Response
	url  string
	body []byte
	err  error
}

func getURLWithTimeout(ctx context.Context, url string, done chan<- Resp, clinet *http.Client) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bu := make([]byte, 0)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		bytes.NewBuffer(bu),
	)
	if err != nil {
		done <- Resp{url: url, err: err}
		return
	}

	resp, err := clinet.Do(req)
	if err != nil {
		done <- Resp{url: url, err: err}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		done <- Resp{url: url, err: err}
		return
	}

	done <- Resp{r: resp, url: url, body: body}
}
