package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	CRLF               = "\r\n"
	HTTPStatusOK       = "HTTP/1.1 200 OK"
	HTTPStatusNotFound = "HTTP/1.1 404 Not Found"
	ContentType        = "Content-Type"
)

func main() {
	fmt.Println("Starting server...")

	listener, err := net.Listen("tcp", "localhost:4221")
	if err != nil {
		fmt.Printf("Failed to bind to port 4221: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Listening on localhost:4221")

	// Accept and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go handleConnection(conn) // Handle each connection concurrently
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the HTTP request from the connection
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Printf("Error reading request: %v\n", err)
		return
	}

	// Log the request method and URL path
	fmt.Printf("Request: %s %s\n", request.Method, request.URL.Path)

	// Respond based on the requested URL path
	switch {
	case request.URL.Path == "/":
		sendResponse(conn, HTTPStatusOK, nil, "")
	case strings.HasPrefix(request.URL.Path, "/echo/"):
		echoString := strings.TrimPrefix(request.URL.Path, "/echo/")
		headers := map[string]string{
			ContentType:      "text/plain",
			"Content-Length": fmt.Sprintf("%d", len(echoString)),
		}
		sendResponse(conn, HTTPStatusOK, headers, echoString)
	case request.URL.Path == "/user-agent":
		userAgent := request.Header.Get("User-Agent")
		headers := map[string]string{
			ContentType:      "text/plain",
			"Content-Length": fmt.Sprintf("%d", len(userAgent)),
		}
		sendResponse(conn, HTTPStatusOK, headers, userAgent)
	default:
		sendResponse(conn, HTTPStatusNotFound, nil, "")
	}
}

func sendResponse(conn net.Conn, status string, headers map[string]string, body string) {
	response := status + CRLF
	if headers != nil {
		for key, value := range headers {
			response += fmt.Sprintf("%s: %s%s", key, value, CRLF)
		}
	}
	response += CRLF + body

	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}
}
