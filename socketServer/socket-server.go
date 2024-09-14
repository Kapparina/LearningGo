package socketServer

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	port     string = "8080"
	response string = "Message received!"
)

func Example() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGKILL,
	)
	defer stop()

	go startServer(ctx)
	<-ctx.Done()
	stop()
	slog.Info("Server stopped")
}

func startServer(ctx context.Context) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", net.JoinHostPort("", port))
	if err != nil {
		slog.Error("Error resolving address", "error", err)
		panic(err)
	}
	listener, err := net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		slog.Error("Error listening", "error", err)
		return
	}
	defer func() {
		if err = listener.Close(); err != nil {
			slog.Error("Error closing listener", "error", err)
		}
	}()
	slog.Info("Server started", "address", listener.Addr(), "protocol", listener.Addr().Network())
	serve(ctx, listener)
}

func serve(ctx context.Context, listener net.Listener) {
	semaphore := make(chan struct{}, 100)

	for {
		select {
		case <-ctx.Done():
			_ = listener.Close()
			slog.Info("Context cancelled")
			return
		default:
			conn, connErr := listener.Accept()
			if connErr != nil {
				slog.Error("Error accepting connection", "connection", connErr)
				continue
			}
			semaphore <- struct{}{}
			slog.Info("Connections", "remaining", cap(semaphore)-len(semaphore))
			go handleConnection(ctx, conn, semaphore)
		}
	}
}

func handleConnection(ctx context.Context, conn net.Conn, semaphore chan struct{}) {
	defer func(conn net.Conn) {
		slog.Info(
			"Connection closed",
			"remote_addr", conn.RemoteAddr(),
			"local_addr", conn.LocalAddr(),
		)
		if err := conn.Close(); err != nil {
			slog.Error("Error closing connection", "error", err)
		}
	}(conn)
	defer func() { <-semaphore }()

	slog.Info("New connection accepted", "remote_addr", conn.RemoteAddr(), "local_addr", conn.LocalAddr())

	var (
		buf = make([]byte, 1024)
		rw  = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	)
readLoop:
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled")
			handleWrite(rw, "Server termination requested, goodbye")
			return
		default:
			idx, err := rw.Read(buf)

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
			}
		}
	}

	handleWrite(rw, response)
}

func handleWrite(writer *bufio.ReadWriter, msg string) {
	if _, err := writer.Write([]byte(msg)); err != nil {
		slog.Error("Error writing data", "error", err)
		return
	}
	if err := writer.Flush(); err != nil {
		slog.Error("Error flushing data", "error", err)
		return
	}
}
