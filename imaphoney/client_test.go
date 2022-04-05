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
		//{"A01 CAPABILITY", "* CAPABILITY IMAP4rev1 AUTH=PLAIN"},
		{"A02 NOOP", "A02 OK"},
		{"A03 LOGIN joe password", "A03 NO LOGIN failed"},
		{"A04 CLOSE", ""},
	}

	s := NewServer("localhost", ":1992",
		"", "", false)
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

	client, hello := NewClient("localhost:1992")
	println("hello from server : ", hello)
	client.Send("A01 CAPABILITY")
	r1 := client.Read()
	if strings.TrimSuffix(r1,"\r\n") != "* CAPABILITY IMAP4rev1 AUTH=PLAIN" {
			t.Errorf("send: \"A01 CAPABILITY\"\n wait: \"* CAPABILITY IMAP4rev1 AUTH=PLAIN\"\n receive: \"%s\"\n", r1)
	}
	r1 = client.Read()
	if strings.TrimSuffix(r1,"\r\n") != "A01 OK CAPABILITY" {
		t.Errorf(" wait: \"A01 OK CAPABILITY\"\n receive: \"%s\"\n", r1)
	}

	for _, tt := range listTests {
		println("write to server: ", tt.message)

		client.Send(tt.message)
		reply := client.Read()

		println(" wait from server: ", tt.response)
		println("reply from server: ", reply)

		if strings.TrimSuffix(reply,"\r\n") != tt.response {
			t.Errorf("send: \"%s\"\n wait: \"%s\"\n receive: \"%s\"\n", tt.message, tt.response, reply)
		}
	}

}

