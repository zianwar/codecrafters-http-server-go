package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		log.Fatalln("Failed to bind to port 4221")
	}

	log.Println("Listening on 4221")

	// listen to incoming connections (blocking)
	conn, err := l.Accept()
	if err != nil {
		log.Fatalf("Error accepting connection: %s", err.Error())
	}

	handleConn(conn)
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// read data from connection
	buffer := make([]byte, 1024) // 1024 is a good start
	_, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading data: %v", err)
	}

	_, rest, ok := strings.Cut(string(buffer), " ")
	if !ok {
		log.Println("Invalid http request")
	}
	path, _, ok := strings.Cut(rest, " ")
	if !ok {
		log.Println("Invalid http request")
	}

	// send response
	response := "HTTP/1.1 400 Not Found\r\n\r\n"
	if path == "/" {
		response = "HTTP/1.1 200 OK\r\n\r\n"
	}
	_, err = fmt.Fprintf(conn, "%s", response)
	if err != nil {
		log.Fatalf("Failed to respond %v", err)
	}
}
