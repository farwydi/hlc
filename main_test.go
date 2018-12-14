package main

import (
	"bufio"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func SplitAt(substring string) func(data []byte, atEOF bool) (advance int, token []byte, err error) {

	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {

		// Return nothing if at end of file and no data passed
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Find the index of the input of the separator substring
		if i := strings.Index(string(data), substring); i >= 0 {
			return i + len(substring), data[0:i], nil
		}

		// If at end of file with data return the data
		if atEOF {
			return len(data), data, nil
		}

		return
	}
}

func TestRequestHandlerP1(t *testing.T) {
	answers, err := os.Open("test_accounts/answers/phase_1_get.answ")
	if err != nil {
		t.Fatal(err)
	}
	defer answers.Close()

	ammo, err := os.Open("test_accounts/ammo/phase_1_get.ammo")
	if err != nil {
		t.Fatal(err)
	}
	defer ammo.Close()

	scanner := bufio.NewScanner(answers)
	scannerAmmo := bufio.NewScanner(ammo)
	scannerAmmo.Split(SplitAt("\r\n\r\n"))
	for scanner.Scan() {
		scannerAmmo.Scan()

		anr := scannerAmmo.Text()
		i := strings.Index(anr, "\n")
		name := anr[:i]
		call := anr[i+1:]
		_ = name

		td := strings.Split(scanner.Text(), "\t")
		_ = td

		ln := fasthttputil.NewInmemoryListener()

		serverCh := make(chan struct{})
		go func() {
			if err := fasthttp.Serve(ln, requestHandler); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			close(serverCh)
		}()

		clientCh := make(chan struct{})
		go func() {
			c, err := ln.Dial()
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			var n int
			if n, err = c.Write([]byte(call)); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			_ = n

			br := bufio.NewReader(c)
			var resp fasthttp.Response
			if err = resp.Read(br); err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			code, _ := strconv.Atoi(td[2])
			if code != resp.StatusCode() {
				t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), code)
			}
			if !resp.ConnectionClose() {
				t.Fatalf("expecting 'Connection: close' response header")
			}
			if string(resp.Body()) != td[3] {
				t.Fatalf("unexpected body: %q. Expecting %q", resp.Body(), td[3])
			}

			close(clientCh)
		}()

		select {
		case <-clientCh:
		case <-time.After(time.Second * 2):
			t.Fatalf("timeout")
		}

		if err := ln.Close(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		select {
		case <-serverCh:
		case <-time.After(time.Second * 2):
			t.Fatalf("timeout")
		}
	}
}

func TestServerResponseServerHeader(t *testing.T) {
	serverName := "foobar serv"

	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			name := ctx.Response.Header.Server()
			if string(name) != serverName {
				fmt.Fprintf(ctx, "unexpected server name: %q. Expecting %q", name, serverName)
			} else {
				ctx.WriteString("OK")
			}

			// make sure the server name is sent to the client after ctx.Response.Reset()
			ctx.NotFound()
		},
		Name: serverName,
	}

	ln := fasthttputil.NewInmemoryListener()

	serverCh := make(chan struct{})
	go func() {
		if err := s.Serve(ln); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(serverCh)
	}()

	clientCh := make(chan struct{})
	go func() {
		c, err := ln.Dial()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if _, err = c.Write([]byte("GET / HTTP/1.1\r\nHost: aa\r\n\r\n")); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		br := bufio.NewReader(c)
		var resp fasthttp.Response
		if err = resp.Read(br); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if resp.StatusCode() != fasthttp.StatusNotFound {
			t.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusNotFound)
		}
		if string(resp.Body()) != "404 Page not found" {
			t.Fatalf("unexpected body: %q. Expecting %q", resp.Body(), "404 Page not found")
		}
		if string(resp.Header.Server()) != serverName {
			t.Fatalf("unexpected server header: %q. Expecting %q", resp.Header.Server(), serverName)
		}
		if err = c.Close(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		close(clientCh)
	}()

	select {
	case <-clientCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	select {
	case <-serverCh:
	case <-time.After(time.Second):
		t.Fatalf("timeout")
	}
}
