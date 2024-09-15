package socketServer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"
)

const (
	response string = "Message received!\n"
)

func Example() {
	// Set up context to listen for OS signals (e.g. SIGTERM, SIGINT) to handle shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
		syscall.SIGINT,
	)
	defer stop()

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(100)

	go func() {
		if err := startServer(ctx, &wg, sem); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("Termination signal received, attempting graceful shutdown")

	// Wait for all active connections to finish before fully shutting down
	wg.Wait()
}

func startServer(ctx context.Context, wg *sync.WaitGroup, sem *semaphore.Weighted) error {
	// Create the TCP listener on the specified port
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: 8080})
	if err != nil {
		return err
	}
	slog.Info("Server started", "address", listener.Addr().String())

	connCh := make(chan net.Conn)
	go acceptConnections(ctx, listener, connCh)

	for {
		select {
		case <-ctx.Done(): // If the context is cancelled, shut down the server
			return listener.Close()
		case conn := <-connCh: // When a new connection is accepted
			if !sem.TryAcquire(1) {
				slog.Info("Server is full, rejecting new connection")
				_ = conn.Close()
				continue
			}
			wg.Add(1) // Track the connection in the WaitGroup
			go func() {
				defer wg.Done() // Mark connection as done after processing
				defer sem.Release(1)
				if err = handleConnection(ctx, 10*time.Second, conn); err != nil {
					slog.Error("Error handling connection", "error", err)
				} else {
					slog.Info("Client disconnected", "remote_addr", conn.RemoteAddr())
				}
			}()
		}
	}
}

func acceptConnections(ctx context.Context, listener net.Listener, connCh chan<- net.Conn) {
	defer close(connCh) // Close the connection channel when done
	for {
		conn, err := listener.Accept() // Accept a new incoming connection
		if err != nil {
			if ctx.Err() != nil { // If the context is cancelled, exit
				slog.Info("Server received shutdown signal, not accepting new connections")
				return
			}
			slog.Error("Error accepting connection", "error", err)
			continue
		}

		// Send the accepted connection to the connection channel
		select {
		case connCh <- conn:
			slog.Info("Connection received", "remote_addr", conn.RemoteAddr())
		case <-ctx.Done(): // Stop accepting new connections if context is cancelled
			return
		}
	}
}

func handleConnection(ctx context.Context, waitTime time.Duration, conn net.Conn) error {
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("Error closing connection", "error", err)
		}
	}()
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	buf := make([]byte, 1024) // Buffer for reading data from the connection
	for {
		// Check if the server is shutting down before processing further
		if errors.Is(ctx.Err(), context.Canceled) {
			return handleWrite(rw, "Termination requested by server, goodbye")
		}
		_ = conn.SetReadDeadline(time.Now().Add(waitTime))

		if n, err := rw.Read(buf); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		} else {
			slog.Info("Received data", "data", string(buf[:n]), "remote_addr", conn.RemoteAddr())
			_ = handleWrite(rw, response)
		}
	}
}

// handleWrite performs a regular write operation to the client with context checks
func handleWrite(writer *bufio.ReadWriter, msg string) error {
	if _, err := writer.Write([]byte(msg)); err != nil || writer.Flush() != nil {
		slog.Error("Error writing data", "error", err)
		return fmt.Errorf("error writing data: %w", err)
	}
	slog.Info("Wrote data", "data", msg)
	return nil
}
