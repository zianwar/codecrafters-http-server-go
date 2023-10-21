package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Request struct {
	method  string
	path    string
	headers map[string]string
	body    string
	proto   string
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
		if newLinesCount == 1 && line == "" && contentLength == 0 {
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
