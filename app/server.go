package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Fatalln("Failed to bind to port 4221")
	}

	log.Println("Listening on 4221")

	conn, err := l.Accept()
	if err != nil {
		log.Fatalf("Error accepting connection: %s", err.Error())
	}

	_, err = fmt.Fprintf(conn, "%s", "HTTP/1.1 200 OK\r\n\r\n")
	if err != nil {
		log.Fatalf("Failed to respond %v", err)
	}

	conn.Close()

}
