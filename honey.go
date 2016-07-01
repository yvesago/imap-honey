package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	debug      bool
	quiet      bool // don't write to console
	addr       string
	hostname   string
	capability string
	listener   net.Listener
	closed     bool
	withTLS    bool
	tlsConfig  *tls.Config
}

func (server *Server) IsDebug() bool {
	return server.debug
}
func (server *Server) SetDebug(d bool) {
	server.debug = d
}
func (server *Server) IsQuiet() bool {
	return server.quiet
}
func (server *Server) SetQuiet(q bool) {
	server.quiet = q
}
func (server *Server) SetCapability(s string) {
	server.capability = s
}
func (server *Server) Closed() bool { return server.closed }
func (server *Server) Close() {
	server.closed = true
	server.listener.Close()
}
func NewServer(hostname string, addr string, certPath string, keyPath string, withTLS bool) *Server {
	var tlsConfig *tls.Config
	if withTLS {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			fmt.Errorf("%v", err)
			return nil
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}

	}
	server := &Server{false, false, addr, hostname, "IMAP4rev1 AUTH=PLAIN", nil, false, withTLS, tlsConfig}
	return server
}

// Session

type Session struct {
	server *Server
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	// Stateful stuff
	state    int
	username string
}

func NewSession(
	server *Server, conn net.Conn,
	reader *bufio.Reader, writer *bufio.Writer,
) *Session {
	s := &Session{server, conn, reader, writer, 0, ""}
	return s
}
func (sess *Session) Sendf(format string, args ...interface{}) {
	fmt.Fprintf(sess.writer, format, args...)
	sess.writer.Flush()
}
func (sess *Session) Readline() (string, error) {
	s, e := sess.reader.ReadString('\n')
	return s, e
}
func (sess *Session) SetUsername(username string) {
	sess.username = username
}
func (sess *Session) RemoteIP() string {
	s := sess.conn.RemoteAddr().String()
	ip, _, _ := net.SplitHostPort(s)
	return ip
}
func (sess *Session) Log(s string) {
	log.Printf(s) // syslog
	if !sess.server.IsQuiet() {
		fmt.Printf("%s - %s\n", time.Now().Format(time.RFC3339), s) // console
	}
}

// Command

type Command struct {
	Tag       string
	Command   string
	Arguments string
}

func ParseCommand(s string) (*Command, error) {
	var tag, com, args string = "", "", ""

	sp := strings.Split(s, " ")

	switch len(sp) {
	case 1:
		args = sp[0]
		if len(args) == 0 {
			return nil, fmt.Errorf("Missing tag in command %q", s)
		}

		lastchar := args[len(args)-1:]
		if lastchar == "=" { // auth plain base64 encoding
			d, err := base64.StdEncoding.DecodeString(args)
			if err == nil {
				d = bytes.Replace(d, []byte("\x00"), []byte(" "), -1)
				args = strconv.QuoteToASCII(string(d[:]))
				com = "LOGIN"
			} else {
				return nil, fmt.Errorf("Missing tag in command %q", s)
			}
		}
	case 2:
		tag = sp[0]
		com = strings.ToUpper(strings.TrimSpace(sp[1]))
	case 3:
		tag = sp[0]
		com = strings.ToUpper(strings.TrimSpace(sp[1]))
		args = sp[2]
	case 4:
		tag = sp[0]
		com = strings.ToUpper(strings.TrimSpace(sp[1]))
		args = sp[2] + " " + sp[3]
	}

	//fmt.Printf("tag %s, com %s, args %s\n", tag,com,args)

	command := &Command{tag, com, args}
	return command, nil
}

func handle_session(sess *Session) error {
	timeout := time.Duration(3) * time.Minute
	sess.conn.SetReadDeadline(time.Now().Add(timeout))

	if sess.server.IsDebug() {
		sess.Log(fmt.Sprintf("IP: %s, OPENED %p", sess.RemoteIP(), sess))
	}

	// Send greeting
	// sess.Sendf("OK %s IMAP4rev1\r\n", sess.server.hostname)
	sess.Sendf("OK IMAP4\r\n")

	var command *Command
	memtag := "" // for AUTH=PLAIN

command:
	s, e := sess.Readline()
	if e != nil {
		goto err
	}
	s = strings.TrimRight(s, "\r\n")
	if sess.server.IsDebug() {
		sess.Log(fmt.Sprintf("IP: %s, COMMAND: %s", sess.RemoteIP(), s))
	}

	command, e = ParseCommand(s)
	if e != nil {
		goto err
	}

	// Handle commands

	switch command.Command {
	case "CAPABILITY":
		//sess.Sendf("* CAPABILITY ACL ID IDLE IMAP4rev1 AUTH=PLAIN\r\n")
		sess.Sendf("* CAPABILITY %s\r\n", sess.server.capability)
		sess.Sendf("%s OK CAPABILITY\r\n", command.Tag)
		goto command
	case "NOOP":
		sess.Sendf("%s OK\r\n", command.Tag)
		goto command
	case "AUTHENTICATE":
		sess.Sendf("+\r\n")
		memtag = command.Tag
		goto command
	case "LOGIN":
		sess.Log(fmt.Sprintf("IP: %s, LOGIN: %s", sess.RemoteIP(), command.Arguments))
		tag := command.Tag
		time.Sleep(3 * time.Second)
		if command.Tag == "" {
			tag = memtag
		}
		sess.Sendf("%s NO LOGIN failed\r\n", tag)
		goto close
	case "LOGOUT":
		sess.Sendf("* BYE %s\r\n", sess.server.hostname)
		sess.Sendf("%s OK LOGOUT\r\n", command.Tag)
		goto close
	default:
		sess.Sendf("%s BAD invalid command\r\n", command.Tag)
		goto command
	}

close:
	sess.conn.Close()
	if sess.server.IsDebug() {
		sess.Log(fmt.Sprintf("CLOSED %p\n", sess))
	}
	return nil

err:
	sess.conn.Close()
	return fmt.Errorf("handle_session: %v", e)
}

// Server

func Listen(server *Server) error {
	var ln net.Listener
	var e error

	if server.withTLS {
		ln, e = tls.Listen("tcp", server.addr, server.tlsConfig)
	} else {
		ln, e = net.Listen("tcp", server.addr)
	}
	if e != nil {
		return e
	} else {
		server.listener = ln
	}
	return nil
}

func Serve(server *Server) error {
	for {
		conn, e := server.listener.Accept()
		if e != nil {
			if server.Closed() {
				break
			}
			fmt.Printf("accept error: %v\n", e)
			return e
		}
		go func(conn_pointer *net.Conn) {
			conn := *conn_pointer
			sess := NewSession(
				server, conn,
				bufio.NewReader(conn), bufio.NewWriter(conn),
			)

			e = handle_session(sess)
			if e != nil {
				fmt.Printf("Serve() ERROR: %v\n", e)
				return
			}

		}(&conn) //goroutine
	}

	return nil
}

/**

USAGE

openssl genrsa -out server.key 2048
openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650

./honey -d -cert server.pem -key server.key -addr :9443 -server server:514

**/
func main() {

	syslogServerFlag := flag.String("server", "", "syslog remote server")
	hostnameFlag := flag.String("hostname", "localhost", "hostname")
	addressFlag := flag.String("addr", ":1993", "ipaddr:port")
	certFlag := flag.String("cert", "", "cert file")
	keyFlag := flag.String("key", "", "cert file")
	capFlag := flag.String("cap", "ACL ID IDLE IMAP4rev1 AUTH=PLAIN", "imap CAPABILITY")
	debugFlag := flag.Bool("d", false, "debug")
	quietFlag := flag.Bool("q", false, "quiet - no msg in console")
	flag.Parse()

	withTls := false
	if *certFlag != "" && *keyFlag != "" {
		withTls = true
	}

	log.SetFlags(0) // remove useless timestamp for syslog
	if *syslogServerFlag == "" {
		logwriter, err := syslog.New(syslog.LOG_NOTICE, "imaphoney")
		if err == nil {
			log.SetOutput(logwriter)
		}
	} else {
		logwriter, err := syslog.Dial("udp", *syslogServerFlag, syslog.LOG_NOTICE, "imaphoney")
		if err == nil {
			log.SetOutput(logwriter)
		} else {
			fmt.Printf("syslog.Dial() ERROR: %v\n", err)
		}
	}

	s := NewServer(*hostnameFlag, *addressFlag,
		*certFlag, *keyFlag, withTls)
	s.SetDebug(*debugFlag)
	s.SetQuiet(*quietFlag)

	s.SetCapability(*capFlag)

	e := Listen(s)
	if e != nil {
		fmt.Printf("Listen() ERROR: %v\n", e)
		return
	}

	Serve(s)
}
