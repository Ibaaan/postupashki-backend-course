package main

import (
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
		fmt.Fprintln(os.Stderr,
			"No URLs provided. Specify at least one URL. Use --help for usage.")
		os.Exit(1)
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

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr,
			"No URLs provided. Specify at least one URL. Use --help for usage.")
		os.Exit(1)
	}

	done := make(chan Resp, len(args))
	client := http.Client{}
	ctx, cancel := context.WithCancel(context.Background())
	for _, v := range args {
		go getURLWithTimeout(ctx, v, done, &client)
	}

	succedded := false
	results := make([]Resp, len(args))

	for i := 0; i < len(args); i++ {
		response := <-done
		if response.r != nil {
			succedded = true
			fmt.Printf("%s %s\nheaders: %v\nbody:%s\n",
				response.r.Proto,
				response.r.Status,
				response.r.Header,
				string(response.body),
			)
			break
		}
		results = append(results, response)

	}
	cancel()

	if !succedded {
		hadTimeout := false

		fmt.Fprintln(os.Stderr, "All requests failed:")
		for _, r := range results {
			var status string
			if errors.Is(r.err, context.DeadlineExceeded) {
				hadTimeout = true
				status = "timed out"
			} else {
				status = "failed"
			}
			fmt.Fprintf(os.Stderr, "  %s: %s (%v)\n", r.url, status, r.err)
		}

		if hadTimeout {
			os.Exit(228)
		} else {
			os.Exit(1)
		}
	}
}

func setTimeout(args []string) []string {
	if len(args) == 1 {
		fmt.Fprintln(os.Stderr, "Missing timeout: expected an integer number of seconds after the flag.")
		os.Exit(1)
	}
	num, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid timeout: expected an integer number of seconds.")
		os.Exit(1)
	}
	timeout = time.Duration(num * int(time.Second))
	return args[2:]
}

func printHelp() {
	fmt.Print(`Usage: hedgedcurl [OPTIONS] URL [URL...]

Options (place before URLs):
  -t, --timeout SECONDS  Request timeout in seconds (default: 15)
  -h, --help             Show this help message and exit

Examples:
  hedgedcurl https://example.com https://example.org
  hedgedcurl -t 5 https://example.com https://example.org
  hedgedcurl --timeout 30 https://example.com
  hedgedcurl --help
`)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
