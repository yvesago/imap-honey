package main

import (
	"bufio"
	//"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"log/syslog"
	"net"
	"net/mail"
	//"strconv"
	"regexp"
	"strings"
	"time"
)

var Version string

type Server struct {
	debug      bool
	quiet      bool // don't write to console
	addr       string
	hostname   string
	capability string
	listener   net.Listener
	closed     bool
	withTLS    bool
	logAuth    bool
	logData    bool
	authOK     bool
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
func NewServer(hostname string, addr string, certPath string, keyPath string, withTLS bool, logAuth bool, logData bool, authOK bool) *Server {
	var tlsConfig *tls.Config
	if withTLS {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			fmt.Errorf(err.Error())
			return nil
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}

	}

	//server := &Server{false, false, addr, hostname, "250-localhost\r\n", nil, false, withTLS, logAuth, logData, authOK, tlsConfig}
	server := &Server{
		debug:      false,
		quiet:      false, // don't write to console
		addr:       addr,
		hostname:   hostname,
		capability: "250-localhost\r\n",
		listener:   nil,
		closed:     false,
		withTLS:    withTLS,
		logAuth:    logAuth,
		logData:    logData,
		authOK:     authOK,
		tlsConfig:  tlsConfig,
	}
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
func (sess *Session) GetUsername() string {
	return sess.username
}
func (sess *Session) RemoteIP() string {
	s := sess.conn.RemoteAddr().String()
	ip, _, _ := net.SplitHostPort(s)
	return ip
}
func (sess *Session) Log(s string) {
	log.Print(s) // syslog
	if !sess.server.IsQuiet() {
		fmt.Printf("%s - %s\n", time.Now().Format(time.RFC3339), s) // console
	}
}

// Command

type Command struct {
	Command   string
	Arguments string
}

func cleanMail(mails string) string {
	emails, _ := mail.ParseAddressList(mails)
	var u []string
	for _, v := range emails {
		u = append(u, v.Address)
	}
	return strings.Join(u, ", ")
}

func ParseCommand(s string) (*Command, error) {

	command := &Command{"", ""}

	matched, _ := regexp.MatchString(`^\.$`, s)
	switch {
	case strings.Contains(s, "EHLO"):
		sp := strings.Split(s, " ")
		command.Command = sp[0]
		command.Arguments = sp[1]
	case strings.Contains(s, "HELO"):
		sp := strings.Split(s, " ")
		command.Command = sp[0]
		command.Arguments = sp[1]
	case strings.Contains(s, "DATA"):
		command.Command = "DATA"
	case matched:
		command.Command = "END"
	case strings.Contains(s, "QUIT"):
		command.Command = "QUIT"
	case strings.Contains(s, "MAIL FROM"):
		sp := strings.Split(s, ":")
		command.Command = "MAIL"
		command.Arguments = cleanMail(sp[1])
	case strings.Contains(s, "RCPT TO"):
		sp := strings.Split(s, ":")
		command.Command = "TO"
		command.Arguments = cleanMail(sp[1])
	case strings.Contains(s, "AUTH LOGIN"):
		command.Command = "AUTH"
	case strings.Contains(s, "STARTTLS"):
		command.Command = "STARTTLS"
	default:
		command.Command = s
	}
	return command, nil
}

func handle_session(sess *Session) error {
	timeout := time.Duration(3) * time.Minute
	sess.conn.SetReadDeadline(time.Now().Add(timeout))

	if sess.server.IsDebug() {
		sess.Log(fmt.Sprintf("IP: %s, OPENED %p", sess.RemoteIP(), sess))
	}

	// Send greeting
	sess.Sendf("220 %s ESMTP ready\r\n", sess.server.hostname)

	var command *Command

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
	case "HELO":
		sp := strings.Split(sess.server.capability, "\r\n")
		sess.Sendf("%s\r\n", sp[0])
		goto command
	case "EHLO":
		sess.Sendf(sess.server.capability)
		goto command
	case "TO":
		if sess.server.logData {
			sess.Sendf("250 Sender ok\r\n")
			goto command
		}
		sess.Sendf("550 <%s>... Denied due to spam list\r\n", command.Arguments)
		goto close
	case "MAIL":
		sess.Sendf("250 Recipient ok\r\n")
		goto command
	case "DATA":
		sess.Sendf("354 Enter mail, end with \".\" on a line by itself\r\n")
		goto command
	case "END":
		sess.Sendf("250 Ok\r\n")
		goto command
	case "RSET":
		sess.Sendf("250 Ok\r\n")
		goto command
	case "QUIT":
		sess.Sendf("221 2.0.0 Bye\r\n")
		goto close
	case "AUTH":
		if sess.server.logAuth {
			sess.Sendf("334 VXNlcm5hbWU6\r\n")
			goto command
		}
		sess.Sendf("503 5.5.1 Error: authentication not enabled\r\n")
		goto close
	case "STARTTLS":
		//sess.Sendf("502 5.5.2 Error: command not recognized\r\n")
		//goto close
		sess.Sendf("454 TLS not available due to temporary reason\r\n")
		goto close
	default:
		rawDecodedText, err := base64.StdEncoding.DecodeString(command.Command)
		if err == nil {
			login := sess.GetUsername()
			if login == "" {
				sess.SetUsername(string(rawDecodedText))
				sess.Sendf("334 UGFzc3dvcmQ6\r\n")
			} else {
				sess.Log(fmt.Sprintf("IP: %s, LOGIN: \"%s\", PASS: \"%s\"", sess.RemoteIP(), login, rawDecodedText))
				time.Sleep(3 * time.Second)
				if sess.server.authOK {
					sess.Sendf("2.7.0 Authentication successful\r\n")
				} else {
					sess.Sendf("535 5.7.0 Error: authentication failed\r\n")
					goto close
				}
			}
		} else {
			sess.Sendf("502 5.5.2 Error: command not recognized\r\n")
			goto close
		}

		//sess.Sendf("BAD invalid command\r\n")
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

	fmt.Printf("Version: %s\n", Version)
	syslogServerFlag := flag.String("server", "", "syslog remote server")
	hostnameFlag := flag.String("hostname", "localhost", "hostname")
	addressFlag := flag.String("addr", ":1993", "ipaddr:port")
	certFlag := flag.String("cert", "", "cert file")
	keyFlag := flag.String("key", "", "cert file")
	var capFlag string
	flag.StringVar(&capFlag, "cap", "250-localhost;250-PIPELINING;250-SIZE 5242880;250-ETRN;250 8BITMIME;250 DSN;", "smtp CAPABILITY")
	logAuthFlag := flag.Bool("la", false, "log auth")
	logDataFlag := flag.Bool("ld", false, "log data")
	authOk := flag.Bool("aok", false, "auth ok")
	debugFlag := flag.Bool("d", false, "debug")
	quietFlag := flag.Bool("q", false, "quiet - no msg in console")
	flag.Parse()

	withTls := false
	if *certFlag != "" && *keyFlag != "" {
		withTls = true
	}

	log.SetFlags(0) // remove useless timestamp for syslog
	if *syslogServerFlag == "" {
		logwriter, err := syslog.New(syslog.LOG_NOTICE, "smtphoney")
		if err == nil {
			log.SetOutput(logwriter)
		}
	} else {
		logwriter, err := syslog.Dial("udp", *syslogServerFlag, syslog.LOG_NOTICE, "smtphoney")
		if err == nil {
			log.SetOutput(logwriter)
		} else {
			fmt.Printf("syslog.Dial() ERROR: %v\n", err)
		}
	}

	s := NewServer(*hostnameFlag, *addressFlag,
		*certFlag, *keyFlag, withTls,
		*logAuthFlag, *logDataFlag, *authOk)
	s.SetDebug(*debugFlag)
	s.SetQuiet(*quietFlag)

	u := strings.ReplaceAll(capFlag, ";", "\r\n")
	s.SetCapability(u)

	e := Listen(s)
	if e != nil {
		fmt.Printf("Listen() ERROR: %v\n", e)
		return
	}

	Serve(s)
}
