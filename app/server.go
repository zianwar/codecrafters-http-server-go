package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	lineTerminator       = "\r\n"
	doubleLineTerminator = "\r\n\r\n"
	port                 = "4221"

	httpProtocol = "HTTP/1.1"

	contentTypeText HttpContentType = "text/plain"

	httpStatusOk       HttpStatus = "200 Ok"
	httpStatusNotFound HttpStatus = "400 Not Found"
)

var (
	flDirectory string
)

func init() {
	flag.StringVar(&flDirectory, "directory", "", "Absolute path to the directory to serve files from")
}

type HttpContentType string
type HttpStatus string

type Request struct {
	method  string
	path    string
	headers map[string]string
	body    string
}

type Response struct {
	status  HttpStatus
	body    string
	headers map[string]string
}

func main() {
	flag.Parse()

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

	statusLine, headers, _ := strings.Cut(statusLineAndHeaders, lineTerminator)

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
	resp, err := handleRequest(req)
	if err != nil {
		log.Println(err)
	}

	// Write response
	_, err = conn.Write([]byte(resp.Raw()))
	if err != nil {
		log.Printf("Failed to write response: %v\n", err)
		return
	}
}

func (r *Response) Raw() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s %s", httpProtocol, r.status) + lineTerminator)
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

func NewResponse(status HttpStatus, headers map[string]string, body string) *Response {
	finalHeaders := map[string]string{}
	for k, v := range headers {
		finalHeaders[k] = v
	}
	resp := &Response{
		status:  status,
		body:    body,
		headers: headers,
	}
	return resp
}

func handleRequest(req *Request) (*Response, error) {
	defaultHeaders := map[string]string{
		"Content-Type": string(contentTypeText),
	}

	if req.path == "/" {
		return NewResponse(httpStatusOk, defaultHeaders, ""), nil
	}
	if strings.HasPrefix(req.path, "/echo/") {
		return handleGetEcho(req)
	}
	if strings.HasPrefix(req.path, "/user-agent") {
		return handleGetUserAgent(req)
	}
	if strings.HasPrefix(req.path, "/files/") {
		return handleGetFile(req)
	}
	return NewResponse(httpStatusNotFound, defaultHeaders, ""), nil
}

func handleGetUserAgent(req *Request) (*Response, error) {
	body := ""
	if v, ok := req.headers["user-agent"]; ok {
		body = v
	}

	headers := map[string]string{
		"Content-Type":   string(contentTypeText),
		"Content-Length": fmt.Sprintf("%d", len(body)),
	}
	return NewResponse(httpStatusOk, headers, body), nil
}

func handleGetEcho(req *Request) (*Response, error) {
	path := strings.TrimPrefix(req.path, "/")
	_, content, _ := strings.Cut(path, "/")

	headers := map[string]string{
		"Content-Type":   string(contentTypeText),
		"Content-Length": fmt.Sprintf("%d", len(content)),
	}
	return NewResponse(httpStatusOk, headers, content), nil
}

func handleGetFile(req *Request) (*Response, error) {
	if flDirectory == "" {
		return nil, fmt.Errorf("missing required directory flag")
	}
	// Check if directory already exists
	if _, err := os.Stat(flDirectory); err != nil {
		return nil, fmt.Errorf("cannot read directory %s", flDirectory)
	}

	path := strings.TrimPrefix(req.path, "/")
	_, filename, _ := strings.Cut(path, "/")
	if filename == "" {
		return nil, fmt.Errorf("missing filename from path")
	}

	fullPath := filepath.Join(flDirectory, filename)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s", fullPath)
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s", fullPath)
	}

	headers := map[string]string{
		"Content-Type":   "application/octet-stream",
		"Content-Length": fmt.Sprintf("%d", len(fileBytes)),
	}
	return NewResponse(httpStatusOk, headers, string(fileBytes)), nil
}
