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
	HTTPStatusCreated      = "HTTP/1.1 201 Created"
	HTTPStatusNotFound     = "HTTP/1.1 404 Not Found"
	ContentTypeHeader      = "Content-Type"
	ContentLengthHeader    = "Content-Length"
	ContentEncodingHeader  = "Content-Encoding"
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
		logError(fmt.Sprintf("Failed to bind to port 4221: %v", err))
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println(LogPrefix + "Listening on localhost:4221")

	// Handle graceful shutdown
	handleGracefulShutdown(listener)

	// Accept and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			logError(fmt.Sprintf("Error accepting connection: %v", err))
			continue
		}
		go handleConnection(conn) // Handle each connection concurrently
	}
}

func handleGracefulShutdown(listener net.Listener) {
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownCh
		fmt.Println(LogPrefix + "Shutting down server...")
		listener.Close()
		os.Exit(0)
	}()
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read the HTTP request from the connection
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		logError(fmt.Sprintf("Error reading request: %v", err))
		return
	}

	// Log the request method and URL path
	fmt.Printf(LogPrefix+"Request: %s %s\n", request.Method, request.URL.Path)

	// Determine the encoding requested by the client
	useGzip := supportsGzipEncoding(request.Header.Get("Accept-Encoding"))

	// Respond based on the requested URL path
	switch {
	case request.URL.Path == "/":
		sendResponse(conn, HTTPStatusOK, nil, "")
	case strings.HasPrefix(request.URL.Path, "/echo/"):
		handleEchoRequest(conn, request, useGzip)
	case request.URL.Path == "/user-agent":
		handleUserAgentRequest(conn, request)
	case strings.HasPrefix(request.URL.Path, "/files/"):
		handleFileRequest(conn, request)
	default:
		sendResponse(conn, HTTPStatusNotFound, nil, "")
	}
}

func handleEchoRequest(conn net.Conn, request *http.Request, useGzip bool) {
	echoString := strings.TrimPrefix(request.URL.Path, "/echo/")
	headers := createHeaders(ContentTypeTextPlain, len(echoString))
	if useGzip {
		headers[ContentEncodingHeader] = "gzip"
	}
	sendResponse(conn, HTTPStatusOK, headers, echoString)
}

func handleUserAgentRequest(conn net.Conn, request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	headers := createHeaders(ContentTypeTextPlain, len(userAgent))
	sendResponse(conn, HTTPStatusOK, headers, userAgent)
}

func handleFileRequest(conn net.Conn, request *http.Request) {
	filename := strings.TrimPrefix(request.URL.Path, "/files/")
	filePath := filepath.Join(fileDirectory, filename)

	switch request.Method {
	case http.MethodGet:
		handleFileGetRequest(conn, filePath)
	case http.MethodPost:
		handleFilePostRequest(conn, request, filePath)
	default:
		sendResponse(conn, HTTPStatusNotFound, nil, "")
	}
}

func handleFileGetRequest(conn net.Conn, filePath string) {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			sendResponse(conn, HTTPStatusNotFound, nil, "")
		} else {
			logError(fmt.Sprintf("Error reading file: %v", err))
		}
		return
	}

	headers := createHeaders(ContentTypeOctetStream, len(fileData))
	sendResponse(conn, HTTPStatusOK, headers, string(fileData))
}

func handleFilePostRequest(conn net.Conn, request *http.Request, filePath string) {
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logError(fmt.Sprintf("Error reading request body: %v", err))
		return
	}

	err = ioutil.WriteFile(filePath, body, 0644)
	if err != nil {
		logError(fmt.Sprintf("Error writing file: %v", err))
		return
	}

	sendResponse(conn, HTTPStatusCreated, nil, "")
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
		logError(fmt.Sprintf("Error writing response: %v", err))
	}
}

func logError(message string) {
	fmt.Println(LogPrefix + message)
}

func supportsGzipEncoding(acceptEncoding string) bool {
	encodings := strings.Split(strings.ToLower(acceptEncoding), ",")
	for _, encoding := range encodings {
		if strings.TrimSpace(encoding) == "gzip" {
			return true
		}
	}
	return false
}
