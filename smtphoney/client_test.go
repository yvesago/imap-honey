package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
)

type Client struct {
	socket net.Conn
}

func (client *Client) Send(msg string) {
	client.socket.Write([]byte(msg + "\r\n"))
}

func (client *Client) Read() string {
	reader := bufio.NewReader(client.socket)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			client.socket.Close()
			return ""
		}
		return string(message)
	}
}

func NewClient(addr string) (*Client, string) {
	connection, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	client := &Client{socket: connection}
	hello := client.Read()
	return client, hello
}

func TestMail(t *testing.T) {
	var listTests = []struct {
		message  string // input
		response string // expected result
	}{
		{"EHLO truc", "250-localhost"},
		{"MAIL FROM: <test@example.org>", "250 Recipient ok"},
		{"RCPT TO: Some One <to@example.org>", "550 <to@example.org>... Denied due to spam list"},
	}

	s := NewServer("localhost", ":1993",
		"", "", false,
		false, false, false)
	//s.SetDebug(true)
	//s.SetQuiet(false)

	//u := strings.ReplaceAll(capFlag, ";", "\r\n")
	//s.SetCapability(u)

	e := Listen(s)
	if e != nil {
		fmt.Printf("Listen() ERROR: %v\n", e)
		return
	}

	go Serve(s)

	client, hello := NewClient("localhost:1993")
	println("hello from server : ", hello)

	for _, tt := range listTests {
		println("write to server: ", tt.message)

		client.Send(tt.message)
		reply := client.Read()

		println(" wait from server: ", tt.response)
		println("reply from server: ", reply)

		if strings.TrimSuffix(reply, "\r\n") != tt.response {
			t.Errorf("send: \"%s\"\n wait: \"%s\"\n receive: \"%s\"\n", tt.message, tt.response, reply)
		}
	}

}

func TestAuth(t *testing.T) {
	var listTests = []struct {
		message  string // input
		response string // expected result
	}{
		{"EHLO truc", "250-localhost"},
		{"AUTH LOGIN", "334 VXNlcm5hbWU6"},
		{"YWRtaW4=", "334 UGFzc3dvcmQ6"},
		{"YWRtaW4=", "535 5.7.0 Error: authentication failed"},
	}

	s := NewServer("localhost", ":1994",
		"", "", false,
		true, false, false)
	//s.SetDebug(true)
	//s.SetQuiet(false)

	//u := strings.ReplaceAll(capFlag, ";", "\r\n")
	//s.SetCapability(u)

	e := Listen(s)
	if e != nil {
		fmt.Printf("Listen() ERROR: %v\n", e)
		return
	}

	go Serve(s)

	client, hello := NewClient("localhost:1994")
	println("hello from server : ", hello)

	for _, tt := range listTests {
		println("write to server: ", tt.message)

		client.Send(tt.message)
		reply := client.Read()

		println(" wait from server: ", tt.response)
		println("reply from server: ", reply)

		if strings.TrimSuffix(reply, "\r\n") != tt.response {
			t.Errorf("send: \"%s\"\n wait: \"%s\"\n receive: \"%s\"\n", tt.message, tt.response, reply)
		}
	}

}

func TestErrors(t *testing.T) {
	var listTests = []struct {
		message  string // input
		response string // expected result
	}{
		{"HELO", "250-localhost"},
		{"EHLO", "250-localhost"},
		{"MAIL FROM: ", "250 Recipient ok"},
		{"RCPT TO: ", "550 <>... Denied due to spam list"},
	}

	s := NewServer("localhost", ":1995",
		"", "", false,
		false, false, false)
	//s.SetDebug(true)
	//s.SetQuiet(false)

	//u := strings.ReplaceAll(capFlag, ";", "\r\n")
	//s.SetCapability(u)

	e := Listen(s)
	if e != nil {
		fmt.Printf("Listen() ERROR: %v\n", e)
		return
	}

	go Serve(s)

	client, hello := NewClient("localhost:1995")
	println("hello from server : ", hello)

	for _, tt := range listTests {
		println("write to server: ", tt.message)

		client.Send(tt.message)
		reply := client.Read()

		println(" wait from server: ", tt.response)
		println("reply from server: ", reply)

		if strings.TrimSuffix(reply, "\r\n") != tt.response {
			t.Errorf("send: \"%s\"\n wait: \"%s\"\n receive: \"%s\"\n", tt.message, tt.response, reply)
		}
	}

}
func TestErrors2(t *testing.T) {
	var listTests = []struct {
		message  string // input
		response string // expected result
	}{
		{"HELO", "250-localhost"},
		{"RCPT TO ", "221 2.0.0 Bye"},
		{"MAIL FROM", ""},
	}

	s := NewServer("localhost", ":1996",
		"", "", false,
		false, false, false)
	//s.SetDebug(true)
	//s.SetQuiet(false)

	//u := strings.ReplaceAll(capFlag, ";", "\r\n")
	//s.SetCapability(u)

	e := Listen(s)
	if e != nil {
		fmt.Printf("Listen() ERROR: %v\n", e)
		return
	}

	go Serve(s)

	client, hello := NewClient("localhost:1996")
	println("hello from server : ", hello)

	for _, tt := range listTests {
		println("write to server: ", tt.message)

		client.Send(tt.message)
		reply := client.Read()

		println(" wait from server: ", tt.response)
		println("reply from server: ", reply)

		if strings.TrimSuffix(reply, "\r\n") != tt.response {
			t.Errorf("send: \"%s\"\n wait: \"%s\"\n receive: \"%s\"\n", tt.message, tt.response, reply)
		}
	}

}
