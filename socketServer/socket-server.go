package socketServer

import (
	"bufio"
	"io"
	"log/slog"
	"net"
)

const (
	port     int    = 8080
	response string = "Message received!"
)

type signal struct{}

func StartServer() {
	tcpAddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("", "8080"))
	if err != nil {
		slog.Error("Error resolving address", "error", err)
		panic(err)
	}
	listener, err := net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = listener.Close()
		if err != nil {
			panic(err)
		}
	}()
	slog.Info("Server started", "address", listener.Addr(), "protocol", listener.Addr().Network())

	semaphore := make(chan signal, 100)

	for {
		conn, connErr := listener.Accept()
		if connErr != nil {
			slog.Error("Error accepting connection: ", "connection", connErr)
			continue
		}
		semaphore <- signal{}
		go handleConnection(conn, semaphore)
	}
}

func handleConnection(conn net.Conn, semaphore chan signal) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)
	defer func() { <-semaphore }()

	slog.Info("New connection accepted", "remote_addr", conn.RemoteAddr(), "local_addr", conn.LocalAddr())

	var (
		buf    = make([]byte, 1024)
		reader = bufio.NewReader(conn)
		writer = bufio.NewWriter(conn)
	)
readLoop:
	for {
		idx, err := reader.Read(buf)

		switch err {
		case nil:
			if idx > 0 {
				data := buf[:idx]
				slog.Info("Received data", "data", string(data))
			}
		case io.EOF:
			break readLoop
		default:
			slog.Error("Error reading data", "error", err)
			return
		}
	}

	if _, err := writer.Write([]byte(response)); err != nil {
		slog.Error("Error writing data", "error", err)
		return
	}
	if err := writer.Flush(); err != nil {
		slog.Error("Error flushing data", "error", err)
	}
	slog.Info(
		"Connection closed",
		"remote_addr", conn.RemoteAddr(),
		"local_addr", conn.LocalAddr(),
		"response", response,
	)
}
