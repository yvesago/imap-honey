# imap-honey

Simple IMAP honeypot written in Golang with log to console or syslog

## Quick start

```
$ go run honey.go &

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


## IMAPS support

1) Create public/private keys via:

```
openssl genrsa -out server.key 2048
openssl req -new -x509 -sha256 -key server.key -out server.pem -days 3650
```

2) Run and test

```
$ go run honey.go -cert server.pem -key server.key -addr :9443 -server my.syslog.server:514 &

$ openssl s_client -connect localhost:9443 -quiet
...
```


## Full usage

```
Usage of ./imaphoney:
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


# AUTHORS

Yves Agostini, `<yvesago@cpan.org>`

# LICENSE AND COPYRIGHT

License : MIT

Copyright 2016 - Yves Agostini
