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
	"sync"
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

	// Use WaitGroup to ensure a graceful shutdown
	var wg sync.WaitGroup

	// Start the server
	go func() {
		if err := startServer(ctx, &wg); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("Server received shutdown signal")

	// Wait for all goroutines to finish
	wg.Wait()
}

func startServer(ctx context.Context, wg *sync.WaitGroup) error {
	listener, err := createListener()
	if err != nil {
		return err
	}
	defer closeListener(listener)

	slog.Info("Server started", "address", listener.Addr(), "protocol", listener.Addr().Network())
	return serve(ctx, listener, wg)
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

func serve(ctx context.Context, listener net.Listener, wg *sync.WaitGroup) error {
	// Semaphore to limit the number of concurrent connections
	semaphore := make(chan struct{}, 100)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, closing listener and exiting serve loop")
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

			select {
			case semaphore <- struct{}{}:
				// Increment WaitGroup counter
				wg.Add(1)
				go func() {
					defer wg.Done() // Decrement when goroutine is finished
					handleConnection(ctx, conn, semaphore)
				}()
			case <-ctx.Done():
				slog.Info("Context cancelled, not accepting new connections")
				return listener.Close()
			}
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
		// Check context cancellation before any major operations
		if ctx.Err() != nil {
			slog.Info("Context cancelled for connection")
			err := handleWrite(ctx, rw, "Server termination requested, goodbye")
			if err != nil {
				slog.Error("Error sending termination message", "error", err)
			}
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
			err := handleWrite(ctx, rw, response)
			if err != nil {
				slog.Error("Error writing response", "error", err)
			}
		}
	}
}

func handleWrite(ctx context.Context, writer *bufio.ReadWriter, msg string) error {
	// Check context cancellation before writing
	if ctx.Err() != nil {
		slog.Info("Context cancelled before writing data")
		return ctx.Err()
	}

	if _, err := writer.Write([]byte(msg)); err != nil {
		slog.Error("Error writing data", "error", err)
		return err
	}

	// Check context cancellation again before flushing
	if ctx.Err() != nil {
		slog.Info("Context cancelled before flushing data")
		return ctx.Err()
	}

	if err := writer.Flush(); err != nil {
		slog.Error("Error flushing data", "error", err)
		return err
	}
	return nil
}

func closeConn(conn net.Conn) {
	// Log connection details before closing
	slog.Info("Closing connection", "remote_addr", conn.RemoteAddr(), "local_addr", conn.LocalAddr())
	if err := conn.Close(); err != nil {
		slog.Error("Error closing connection", "error", err)
	}
}
