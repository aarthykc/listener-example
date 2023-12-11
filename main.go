package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	l, err := net.Listen("tcp", ":"+port)
	panicIf(err, "failed to listen")
	fmt.Println("listening on " + l.Addr().String())
	defer l.Close()

	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGUSR1)
	defer done()

	conns := make(chan net.Conn)
	go func() {
		for {
			conn, err := l.Accept()
			// detect if shutdown
			if errors.Is(err, net.ErrClosed) {
				fmt.Println("shutdown listener")
				return
			}
			panicIf(err, "failed to accept")
			fmt.Println("connection from " + conn.RemoteAddr().String())
			conns <- conn
		}
	}()

	wg := sync.WaitGroup{}
	cont := true
	for cont {
		select {
		case <-ctx.Done():
			l.Close()
			cont = false
			continue
		case conn := <-conns:
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendDNSResponse(conn)
			}()
		}
	}
	fmt.Println("waiting for connections to close")
	wg.Wait()
	fmt.Println("shutting down")
}

func panicIf(err error, msg string, args ...interface{}) {
	if err != nil {
		msg = fmt.Sprintf(msg, args...)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func sendDNSResponse(conn net.Conn) {
	defer conn.Close()
	response := `; <<>> DiG 9.10.6 <<>> google.com
	;; global options: +cmd
	;; Got answer:
	;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 41967
	;; flags: qr rd ra; QUERY: 1, ANSWER: 6, AUTHORITY: 0, ADDITIONAL: 1
	
	;; OPT PSEUDOSECTION:
	; EDNS: version: 0, flags:; udp: 512
	;; QUESTION SECTION:
	;google.com.			IN	A
	
	;; ANSWER SECTION:
	google.com.		264	IN	A	142.251.167.102
	google.com.		264	IN	A	142.251.167.101
	google.com.		264	IN	A	142.251.167.113
	google.com.		264	IN	A	142.251.167.139
	google.com.		264	IN	A	142.251.167.138
	google.com.		264	IN	A	142.251.167.100
	
	;; Query time: 135 msec
	;; SERVER: 192.168.1.1#53(192.168.1.1)
	;; WHEN: Mon Dec 11 12:55:21 EST 2023
	;; MSG SIZE  rcvd: 135`

	_, err := conn.Write([]byte(response))
	panicIf(err, "failed to write response")
}
