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
	go startServer(ctx)

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("Server received shutdown signal")
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
			slog.Info("Context cancelled, closing listener")
			return
		default:
			// Accept new connections
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					slog.Info("Context cancelled while accepting connection")
					return
				default:
					slog.Error("Error accepting connection", "error", err)
					continue
				}
			}
			slog.Info(
				"New connection accepted",
				"remote_addr", conn.RemoteAddr(),
				"local_addr", conn.LocalAddr(),
			)
			semaphore <- struct{}{}
			slog.Info("Connections remaining", "remaining", cap(semaphore)-len(semaphore))
			go handleConnection(ctx, conn, semaphore)
		}
	}
}

func handleConnection(ctx context.Context, conn net.Conn, semaphore chan struct{}) {
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("Error closing connection", "error", err)
		}
		slog.Info(
			"Connection closed",
			"remote_addr", conn.RemoteAddr(),
			"local_addr", conn.LocalAddr(),
		)
		<-semaphore
	}()
	var (
		buf = make([]byte, 1024)
		rw  = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled for connection")
			_ = handleWrite(ctx, rw, "Server termination requested, goodbye")
			return
		default:
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			idx, err := rw.Read(buf)
			if err != nil {
				if err == io.EOF {
					slog.Info("Client disconnected", "remote_addr", conn.RemoteAddr())
					return
				}
				slog.Error("Error reading data", "error", err)
				return
			}
			if idx > 0 {
				data := buf[:idx]
				slog.Info("Received data", "data", string(data))
				_ = handleWrite(ctx, rw, response)
			}
		}
	}
}

func handleWrite(ctx context.Context, writer *bufio.ReadWriter, msg string) error {
	select {
	case <-ctx.Done():
		slog.Info("Context cancelled before writing data")
		return ctx.Err()
	default:
		// Try to write the message
		if _, err := writer.Write([]byte(msg)); err != nil {
			slog.Error("Error writing data", "error", err)
			return err
		}
		// Try to flush the writer
		if err := writer.Flush(); err != nil {
			slog.Error("Error flushing data", "error", err)
			return err
		}
	}
	return nil
}
