package socketServer

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	port     string = "8080"
	response string = "Message received!"
)

func Example() {
	// Set up context for handling OS signals
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGINT,
	)
	defer stop()

	// Start the server
	go func() {
		if err := startServer(ctx); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("Server received shutdown signal")
}

func startServer(ctx context.Context) error {
	listener, err := createListener()
	if err != nil {
		return err
	}
	defer closeListener(listener)

	slog.Info("Server started", "address", listener.Addr(), "protocol", listener.Addr().Network())
	return serve(ctx, listener)
}

func createListener() (net.Listener, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", net.JoinHostPort("", port))
	if err != nil {
		slog.Error("Error resolving address", "error", err)
		return nil, err
	}

	listener, err := net.ListenTCP("tcp4", tcpAddr)
	if err != nil {
		slog.Error("Error listening", "error", err)
		return nil, err
	}
	return listener, nil
}

func closeListener(listener net.Listener) {
	if err := listener.Close(); err != nil {
		slog.Error("Error closing listener", "error", err)
	}
}

func serve(ctx context.Context, listener net.Listener) error {
	semaphore := make(chan struct{}, 100)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, closing listener")
			return listener.Close()
		default:
			conn, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					slog.Info("Context cancelled while accepting connection")
					return ctx.Err()
				}
				slog.Error("Error accepting connection", "error", err)
				continue
			}

			slog.Info("New connection accepted", "remote_addr", conn.RemoteAddr(), "local_addr", conn.LocalAddr())
			semaphore <- struct{}{}
			slog.Info("Connections remaining", "remaining", cap(semaphore)-len(semaphore))
			go handleConnection(ctx, conn, semaphore)
		}
	}
}

func handleConnection(ctx context.Context, conn net.Conn, semaphore chan struct{}) {
	defer func() {
		closeConn(conn)
		<-semaphore
	}()

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	buf := make([]byte, 1024)

	for {
		if ctx.Err() != nil {
			slog.Info("Context cancelled for connection")
			_ = handleWrite(ctx, rw, "Server termination requested, goodbye")
			return
		}

		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			slog.Error("Error setting read deadline", "error", err)
			return
		}

		n, err := rw.Read(buf)
		if err != nil {
			if err == io.EOF {
				slog.Info("Client disconnected", "remote_addr", conn.RemoteAddr())
				return
			}
			var nErr net.Error
			if errors.As(err, &nErr) && nErr.Timeout() {
				continue
			}
			slog.Error("Error reading data", "error", err)
			return
		}

		if n > 0 {
			data := buf[:n]
			slog.Info("Received data", "data", string(data))
			_ = handleWrite(ctx, rw, response)
		}
	}
}

func handleWrite(ctx context.Context, writer *bufio.ReadWriter, msg string) error {
	if ctx.Err() != nil {
		slog.Info("Context cancelled before writing data")
		return ctx.Err()
	}

	if _, err := writer.Write([]byte(msg)); err != nil {
		slog.Error("Error writing data", "error", err)
		return err
	}
	if err := writer.Flush(); err != nil {
		slog.Error("Error flushing data", "error", err)
		return err
	}
	return nil
}

func closeConn(conn net.Conn) {
	if err := conn.Close(); err != nil {
		slog.Error("Error closing connection", "error", err)
	}
	slog.Info("Connection closed", "remote_addr", conn.RemoteAddr(), "local_addr", conn.LocalAddr())
}
