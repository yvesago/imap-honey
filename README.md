# imap-honey

Simple IMAP or SMTP honeypot written in Golang with log to console or syslog

## Quick start

```
$ go run imaphoney/honey.go &

$ telnet localhost 1993
Trying ::1...
Connected to localhost.
Escape character is '^]'.
OK IMAP4
a0001 CAPABILITY
* CAPABILITY ACL ID IDLE IMAP4rev1 AUTH=PLAIN
a0001 OK CAPABILITY
a0002 LOGOUT
* BYE localhost
a0002 OK LOGOUT
Connection closed by foreign host.
```

```
$ go run smtphoney/honey.go &

$ telnet localhost 1993
Trying ::1...
Connected to localhost.
Escape character is '^]'.
220 localhost ESMTP ready
EHLO honey
250-localhost
250-PIPELINING
250-SIZE 5242880
250-ETRN
250 8BITMIME
250 DSN
MAIL FROM: test@example.org
250 Recipient ok
QUIT
221 2.0.0 Bye
Connection closed by foreign host.
```

## IMAPS/SMTPS support

1) Create public/private keys via:

```
openssl genrsa -out server.key 2048
openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650
```

2) Run and test

```
$ make all
$ ./build/linux/imaphoney -cert server.pem -key server.key -addr :9443 -server my.syslog.server:514 &

$ openssl s_client -connect localhost:9443 -quiet
...
```


## Full usage

```
Usage of ./build/linux/imaphoney:
  -addr string
        ipaddr:port (default ":1993")
  -cap string
        imap CAPABILITY (default "ACL ID IDLE IMAP4rev1 AUTH=PLAIN")
  -cert string
        cert file
  -d    debug
  -hostname string
        hostname (default "localhost")
  -key string
        cert file
  -q    quiet - no msg in console
  -server string
        syslog remote server
```

```
Usage of ./build/linux/smtphoney:
  -addr string
    	ipaddr:port (default ":1993")
  -aok
    	auth ok
  -cap string
    	smtp CAPABILITY (default "250-localhost;250-PIPELINING;250-SIZE 5242880;250-ETRN;250 8BITMIME;250 DSN;")
  -cert string
    	cert file
  -d	debug
  -hostname string
    	hostname (default "localhost")
  -key string
    	cert file
  -la
    	log auth
  -ld
    	log data
  -q	quiet - no msg in console
  -server string
    	syslog remote server
```

# AUTHORS

Yves Agostini, `<yvesago@cpan.org>`

# LICENSE AND COPYRIGHT

License : MIT

Copyright 2022 - Yves Agostini
