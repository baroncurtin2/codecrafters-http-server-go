package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
)

const (
	CRLF = "\r\n"
)

func Handler(conn net.Conn) {
	defer conn.Close()

	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Println("error reading request: ", err.Error())
		return
	}

	fmt.Printf("Request: %s %s\n", request.Method, request.URL.Path)

	if request.URL.Path != "/" {
		NotFoundResponse(conn)
		return
	}

	OKResponse(conn)
}

func OKResponse(conn net.Conn) {
	_, err := conn.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK%s%s", CRLF, CRLF)))

	if err != nil {
		fmt.Println("connection error:", err.Error())
	}
}

func NotFoundResponse(conn net.Conn) {
	_, err := conn.Write([]byte(fmt.Sprintf("HTTP/1.1 404 Not Found%s%s", CRLF, CRLF)))

	if err != nil {
		fmt.Println("connection error:", err.Error())
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// Uncomment this block to pass the first stage

	l, err := net.Listen("tcp", "localhost:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	Handler(conn)
}
