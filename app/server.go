package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	CRLF                   = "\r\n"
	HTTPStatusOK           = "HTTP/1.1 200 OK"
	HTTPStatusNotFound     = "HTTP/1.1 404 Not Found"
	ContentTypeHeader      = "Content-Type"
	ContentLengthHeader    = "Content-Length"
	ContentTypeTextPlain   = "text/plain"
	ContentTypeOctetStream = "application/octet-stream"
	LogPrefix              = "[SERVER] "
)

var fileDirectory string

func main() {
	// Parse command-line flags
	flag.StringVar(&fileDirectory, "directory", ".", "Directory to serve files from")
	flag.Parse()

	fmt.Println(LogPrefix + "Starting server...")

	listener, err := net.Listen("tcp", "localhost:4221")
	if err != nil {
		fmt.Printf(LogPrefix+"Failed to bind to port 4221: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println(LogPrefix + "Listening on localhost:4221")

	// Handle graceful shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownCh
		fmt.Println(LogPrefix + "Shutting down server...")
		listener.Close()
		os.Exit(0)
	}()

	// Accept and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf(LogPrefix+"Error accepting connection: %v\n", err)
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
		fmt.Printf(LogPrefix+"Error reading request: %v\n", err)
		return
	}

	// Log the request method and URL path
	fmt.Printf(LogPrefix+"Request: %s %s\n", request.Method, request.URL.Path)

	// Respond based on the requested URL path
	switch {
	case request.URL.Path == "/":
		sendResponse(conn, HTTPStatusOK, nil, "")
	case strings.HasPrefix(request.URL.Path, "/echo/"):
		echoString := strings.TrimPrefix(request.URL.Path, "/echo/")
		headers := createHeaders(ContentTypeTextPlain, len(echoString))
		sendResponse(conn, HTTPStatusOK, headers, echoString)
	case request.URL.Path == "/user-agent":
		userAgent := request.Header.Get("User-Agent")
		headers := createHeaders(ContentTypeTextPlain, len(userAgent))
		sendResponse(conn, HTTPStatusOK, headers, userAgent)
	case strings.HasPrefix(request.URL.Path, "/files/"):
		filename := strings.TrimPrefix(request.URL.Path, "/files/")
		handleFileRequest(conn, filename)
	default:
		sendResponse(conn, HTTPStatusNotFound, nil, "")
	}
}

func createHeaders(contentType string, contentLength int) map[string]string {
	return map[string]string{
		ContentTypeHeader:   contentType,
		ContentLengthHeader: fmt.Sprintf("%d", contentLength),
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
		fmt.Printf(LogPrefix+"Error writing response: %v\n", err)
	}
}

func handleFileRequest(conn net.Conn, filename string) {
	filePath := filepath.Join(fileDirectory, filename)
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			sendResponse(conn, HTTPStatusNotFound, nil, "")
		} else {
			fmt.Printf(LogPrefix+"Error reading file: %v\n", err)
		}
		return
	}

	headers := createHeaders(ContentTypeOctetStream, len(fileData))
	sendResponse(conn, HTTPStatusOK, headers, string(fileData))
}
