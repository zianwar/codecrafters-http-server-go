package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	port = "4221"

	lineTerminator = "\r\n"

	contentTypeTextPlain              HttpContentType = "text/plain"
	contentTypeApplicationOctetStream HttpContentType = "application/octet-stream"

	httpStatusOk       HttpStatus = "200 Ok"
	httpStatusCreated  HttpStatus = "201 Ok"
	httpStatusNotFound HttpStatus = "404 Not Found"
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
	proto   string
}

type Response struct {
	status  HttpStatus
	body    string
	headers map[string]string
	proto   string
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

func NewRequest(conn net.Conn) (*Request, error) {
	req := &Request{headers: map[string]string{}}
	scanner := bufio.NewScanner(conn)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		advance, token, err := bufio.ScanLines(data, atEOF)
		if i := bytes.IndexByte(data, '\n'); i < 0 {
			return 0, data, bufio.ErrFinalToken
		}
		return advance, token, err
	})

	var i, newLinesCount, bodyLength, contentLength int
	for scanner.Scan() {
		line := scanner.Text()

		// Parse first line (status line)
		if i == 0 {
			statusLine := strings.Split(line, " ")
			if len(line) < 3 {
				return nil, fmt.Errorf("invalid status line")
			}
			req.method = statusLine[0]
			req.path = statusLine[1]
			req.proto = statusLine[2]
		}

		if line == "" {
			newLinesCount++
		}

		// Parse a single request header as long as
		// we are past first line and haven't found an empty line yet
		if i > 0 && newLinesCount == 0 {
			parts := strings.SplitN(line, ":", 2)
			k := strings.ToLower(strings.TrimSpace(parts[0]))
			v := strings.TrimSpace(parts[1])
			req.headers[k] = v
			if strings.ToLower(k) == "content-length" {
				v, err := strconv.Atoi(v)
				if err != nil {
					return nil, fmt.Errorf("failed to parse content length header")
				}
				contentLength = v
			}
		}

		// Request has no body
		if newLinesCount == 1 && line == "" {
			break
		}

		// Parse request body
		// As long as we reached an empty line and current line read is not empty
		if newLinesCount == 1 && line != "" {
			req.body += line
			bodyLength += len(line) + 1
		}
		if newLinesCount > 1 {
			bodyLength += 1
			req.body += "\n" + line
		}

		if bodyLength != 0 && bodyLength == contentLength {
			break
		}
		i++
	}

	return req, nil
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// Read and parse request
	req, err := NewRequest(conn)
	if err != nil {
		log.Printf("failed to parse request: %v\n", err)
		return
	}

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

	sb.WriteString(fmt.Sprintf("%s %s%s", r.proto, r.status, lineTerminator))
	if len(r.headers) > 0 {
		for k, v := range r.headers {
			sb.WriteString(fmt.Sprintf("%s: %s%s", k, v, lineTerminator))
		}
	}
	sb.WriteString(lineTerminator)
	if len(r.body) > 0 {
		sb.Write([]byte(r.body))
	}
	return sb.String()
}

func NewResponse(status HttpStatus, proto string, headers map[string]string, body string) *Response {
	finalHeaders := map[string]string{}
	for k, v := range headers {
		finalHeaders[k] = v
	}
	resp := &Response{
		status:  status,
		body:    body,
		headers: headers,
		proto:   proto,
	}
	return resp
}

func handleRequest(req *Request) (*Response, error) {
	defaultHeaders := map[string]string{
		"Content-Type": string(contentTypeTextPlain),
	}

	if req.path == "/" {
		return NewResponse(httpStatusOk, req.proto, defaultHeaders, ""), nil
	}

	if req.method == "GET" {
		if strings.HasPrefix(req.path, "/echo/") {
			return handleGetEcho(req)
		}
		if strings.HasPrefix(req.path, "/user-agent") {
			return handleGetUserAgent(req)
		}
		if strings.HasPrefix(req.path, "/files/") {
			return handleGetFile(req)
		}
	}
	if req.method == "POST" {
		if strings.HasPrefix(req.path, "/files/") {
			return handlePostFile(req)
		}
	}

	return NewResponse(httpStatusNotFound, req.proto, defaultHeaders, ""), nil
}

func handleGetUserAgent(req *Request) (*Response, error) {
	body := ""
	if v, ok := req.headers["user-agent"]; ok {
		body = v
	}

	headers := map[string]string{
		"Content-Type":   string(contentTypeTextPlain),
		"Content-Length": fmt.Sprintf("%d", len(body)),
	}
	return NewResponse(httpStatusOk, req.proto, headers, body), nil
}

func handleGetEcho(req *Request) (*Response, error) {
	path := strings.TrimPrefix(req.path, "/")
	_, content, _ := strings.Cut(path, "/")

	headers := map[string]string{
		"Content-Type":   string(contentTypeTextPlain),
		"Content-Length": fmt.Sprintf("%d", len(content)),
	}
	return NewResponse(httpStatusOk, req.proto, headers, content), nil
}

func handleGetFile(req *Request) (*Response, error) {
	if flDirectory == "" {
		return nil, fmt.Errorf("missing required directory flag")
	}

	path := strings.TrimPrefix(req.path, "/")
	_, filename, _ := strings.Cut(path, "/")
	if filename == "" {
		return nil, fmt.Errorf("missing filename from request path")
	}

	fileBytes, err, ok := getFileContent(filename)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	if !ok {
		// file not found
		return NewResponse(httpStatusNotFound, req.proto, headers, ""), nil
	}

	// file found
	headers["Content-Type"] = string(contentTypeApplicationOctetStream)
	headers["Content-Length"] = fmt.Sprintf("%d", len(fileBytes))

	return NewResponse(httpStatusOk, req.proto, headers, string(fileBytes)), nil
}

func handlePostFile(req *Request) (*Response, error) {
	if flDirectory == "" {
		return nil, fmt.Errorf("missing required directory flag")
	}

	path := strings.TrimPrefix(req.path, "/")
	_, filename, _ := strings.Cut(path, "/")
	if filename == "" {
		return nil, fmt.Errorf("missing filename from request path")
	}

	if err := writeFileContent(filename, req.body); err != nil {
		return nil, err
	}

	headers := map[string]string{}
	return NewResponse(httpStatusCreated, req.proto, headers, ""), nil
}

func getFileContent(filename string) ([]byte, error, bool) {
	path := filepath.Join(flDirectory, filename)

	// Return false if file doesn't exist
	if _, err := os.Stat(path); err != nil {
		return nil, nil, false
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s", path), false
	}
	defer file.Close()

	// Read file
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s", path), false
	}

	return fileBytes, nil, true
}

func writeFileContent(filename string, content string) error {
	path := filepath.Join(flDirectory, filename)

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", path, err)
	}
	defer file.Close()

	// Write to file
	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("unable to write file %s", path)
	}

	return nil
}
