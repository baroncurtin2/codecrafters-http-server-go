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
	CRLF                 string = "\r\n"
	HTTPStatusOK         string = "HTTP/1.1 200 OK"
	HTTPStatusNotFound   string = "HTTP/1.1 404 Not Found"
	ContentTypeTextPlain string = "Content-Type: text/plain"
)

func main() {
	fmt.Println("Starting server...")

	listener, err := net.Listen("tcp", "localhost:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221:", err.Error())
		os.Exit(1)
	}

	defer listener.Close()
	fmt.Println("Listening on localhost:4221")

	// accept and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn) // handle each connection concurrently
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// read the http request from the connection
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	// log the request method and url path
	fmt.Printf("Request: %s %s\n", request.Method, request.URL.Path)

	// respond based on the requested URL path
	if request.URL.Path == "/" {
		sendResponse(conn, HTTPStatusOK)
	} else if strings.HasPrefix(request.URL.Path, "/echo/") {
		echoString := strings.TrimPrefix(request.URL.Path, "/echo/")
		sendEchoResponse(conn, echoString)
	} else {
		sendNotFoundResponse(conn)
	}
}

func sendEchoResponse(conn net.Conn, echoString string) {
	headers := fmt.Sprintf("%s%s%s %s%s: %d%s%s", HTTPStatusOK, CRLF, ContentTypeTextPlain, CRLF, "Content-Length", len(echoString), CRLF, CRLF)

	response := headers + echoString
	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func sendResponse(conn net.Conn, status string) {
	response := fmt.Sprintf("%s%s%s", status, CRLF, CRLF)

	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func sendNotFoundResponse(conn net.Conn) {
	sendResponse(conn, HTTPStatusNotFound)
}
