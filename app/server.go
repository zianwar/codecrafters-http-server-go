package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

const (
	lineTerminator = "\r\n"
	port           = "4221"
)

type Request struct {
	method string
	path   string
}

type Response struct {
	status  string
	body    string
	headers map[string]string
}

func main() {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalln("Failed to bind to port " + port)
	}
	log.Println("Listening on " + port)

	// listen to incoming connections
	conn, err := l.Accept()
	if err != nil {
		log.Fatalf("Error accepting connection: %s", err.Error())
	}

	handleConn(conn)
}

func parseRequest(conn net.Conn) (*Request, error) {
	lines := readRequestLines(conn)
	method, path, ok := parseStatusLineParts(lines[0])
	if !ok {
		return nil, fmt.Errorf("failed to parse request status line %s", lines[0])
	}
	req := &Request{
		method: method,
		path:   path,
	}
	return req, nil
}

func readRequestLines(conn net.Conn) []string {
	lines := []string{}
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
		lines = append(lines, line)
	}
	return lines
}

func parseStatusLineParts(statusLine string) (string, string, bool) {
	method, rest, ok1 := strings.Cut(statusLine, " ")
	path, _, ok2 := strings.Cut(rest, " ")
	if !ok1 || !ok2 {
		return "", "", false
	}
	return method, path, true
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// Read and parse request
	req, err := parseRequest(conn)
	if err != nil {
		log.Printf("failed to parse request: %v\n", err)
		return
	}

	// fmt.Println("Request:")
	// fmt.Printf("%q\n", req)

	// Handle request and build a response
	resp := handleRequest(req)
	rawResp := resp.Raw()

	// fmt.Println("Response:")
	// fmt.Printf("%q\n", rawResp)

	// Write response
	_, err = conn.Write([]byte(rawResp))
	if err != nil {
		log.Printf("Failed to write response: %v\n", err)
		return
	}
}

func (r *Response) Raw() string {
	var sb strings.Builder

	sb.WriteString(r.status + lineTerminator)
	if len(r.headers) > 0 {
		for k, v := range r.headers {
			sb.WriteString(fmt.Sprintf("%s: %s", k, v) + lineTerminator)
		}
	}
	sb.WriteString(lineTerminator)
	if len(r.body) > 0 {
		sb.Write([]byte(r.body))
	}
	return sb.String()
}

func handleRequest(req *Request) Response {
	if req.path == "/" {
		return Response{status: "HTTP/1.1 200 OK"}
	}
	if strings.HasPrefix(req.path, "/echo/") {
		return handleEcho(req.path)
	}
	return Response{status: "HTTP/1.1 404 Not Found"}
}

func handleEcho(path string) Response {
	path = strings.TrimPrefix(path, "/")
	_, content, _ := strings.Cut(path, "/")

	return Response{
		status: "HTTP/1.1 200 OK",
		body:   content,
		headers: map[string]string{
			"Content-Type":   "text/plain",
			"Content-Length": fmt.Sprintf("%d", len(content)),
		}}
}
