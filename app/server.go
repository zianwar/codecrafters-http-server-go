package main

import (
	"flag"
	"fmt"
	"log"
	"net"
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

type HttpContentType string
type HttpStatus string

func init() {
	flag.StringVar(&flDirectory, "directory", "", "Absolute path to the directory to serve files from")
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

func handleRequest(req *Request) (*Response, error) {
	if req.path == "/" {
		return NewResponse(httpStatusOk, req.proto, nil, ""), nil
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

	return NewResponse(httpStatusNotFound, req.proto, nil, ""), nil
}

// Handles GET /user-agent [USER-AGENT-HEADER]
// Returns the value of the User-Agent header specified in the request.
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

// Handles GET /echo/<content>
// Returns <content> in response body
func handleGetEcho(req *Request) (*Response, error) {
	path := strings.TrimPrefix(req.path, "/")
	_, content, _ := strings.Cut(path, "/")

	headers := map[string]string{
		"Content-Type":   string(contentTypeTextPlain),
		"Content-Length": fmt.Sprintf("%d", len(content)),
	}
	return NewResponse(httpStatusOk, req.proto, headers, content), nil
}

// Handles GET /files/<filename>
// Returns content of <filename> in response body
func handleGetFile(req *Request) (*Response, error) {
	if flDirectory == "" {
		return nil, fmt.Errorf("missing required directory flag")
	}

	path := strings.TrimPrefix(req.path, "/")
	_, filename, _ := strings.Cut(path, "/")
	if filename == "" {
		return nil, fmt.Errorf("missing filename from request path %s", path)
	}

	fileBytes, err, ok := getFileContent(filename)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	if !ok {
		// file not found, return 404
		return NewResponse(httpStatusNotFound, req.proto, headers, ""), nil
	}

	// file found, set headers and returns 200 along with file content
	headers["Content-Type"] = string(contentTypeApplicationOctetStream)
	headers["Content-Length"] = fmt.Sprintf("%d", len(fileBytes))

	return NewResponse(httpStatusOk, req.proto, headers, string(fileBytes)), nil
}

// Handles POST /files/<filename> <CONTENT_BODY>
// Stores the provided <filename> to disk
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
