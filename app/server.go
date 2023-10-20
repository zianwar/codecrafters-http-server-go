package main

import (
	"fmt"
	"log"
	"net"
	"strings"
)

const (
	lineTerminator       = "\r\n"
	doubleLineTerminator = "\r\n\r\n"
	port                 = "4221"
)

type Request struct {
	method  string
	path    string
	headers map[string]string
	body    string
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

	for {
		// listen to incoming connections
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Error accepting connection: %s", err.Error())
		}

		go handleConn(conn)
	}
}

func readRequest(conn net.Conn) (*Request, error) {
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return NewRequest(string(buffer[0:n]))
}

func NewRequest(s string) (*Request, error) {
	req := &Request{headers: map[string]string{}}

	statusLineAndHeaders, body, ok := strings.Cut(s, doubleLineTerminator)
	if !ok {
		return nil, fmt.Errorf("invalid request 1")
	}
	fmt.Printf("statusLineAndHeaders: %q\n", statusLineAndHeaders)
	fmt.Printf("body: %q\n", body)

	statusLine, headers, _ := strings.Cut(statusLineAndHeaders, lineTerminator)

	fmt.Printf("statusLine: %q\n", statusLine)
	fmt.Printf("headers: %q\n", headers)
	fmt.Printf("body: %q\n", body)

	// Parse status line (first line)
	method, path, ok := parseStatusLineParts(statusLine)
	if !ok {
		return nil, fmt.Errorf("failed to parse request status line %s", statusLine)
	}
	req.method = method
	req.path = path

	// Parse headers
	if len(headers) > 0 {
		for _, h := range strings.Split(headers, lineTerminator) {
			k, v, _ := strings.Cut(h, ":")
			req.headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
		}
	}

	// Parse body
	if len(body) > 0 {
		fmt.Println("body:", body)
		req.body = body
	}

	return req, nil
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
	req, err := readRequest(conn)
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
	defaultHeaders := map[string]string{
		"Content-Type": "text/plain",
	}

	if req.path == "/" {
		return Response{status: "HTTP/1.1 200 OK", headers: defaultHeaders}
	}
	if strings.HasPrefix(req.path, "/echo/") {
		return handleGetEcho(req)
	}
	if strings.HasPrefix(req.path, "/user-agent") {
		return handleGetUserAgent(req)
	}
	return Response{status: "HTTP/1.1 404 Not Found", headers: defaultHeaders}
}

func handleGetUserAgent(req *Request) Response {
	body := ""
	if v, ok := req.headers["user-agent"]; ok {
		body = v
	}

	return Response{
		status: "HTTP/1.1 200 OK",
		body:   body,
		headers: map[string]string{
			"Content-Type":   "text/plain",
			"Content-Length": fmt.Sprintf("%d", len(body)),
		},
	}
}

func handleGetEcho(req *Request) Response {
	path := strings.TrimPrefix(req.path, "/")
	_, content, _ := strings.Cut(path, "/")

	return Response{
		status: "HTTP/1.1 200 OK",
		body:   content,
		headers: map[string]string{
			"Content-Type":   "text/plain",
			"Content-Length": fmt.Sprintf("%d", len(content)),
		}}
}
