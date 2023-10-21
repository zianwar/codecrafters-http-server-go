package main

import (
	"fmt"
	"strings"
)

type Response struct {
	status  HttpStatus
	body    string
	headers map[string]string
	proto   string
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
		headers: finalHeaders,
		proto:   proto,
	}
	return resp
}
