// This file contains tests that are executed to verify your solution.
// It's read-only, so all modifications will be ignored.
package backend

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	commonHTTP "eats/backend/common/http"
)

func newTestSvc(t *testing.T, shutdownTimeout time.Duration) (Svc, *echo.Echo) {
	t.Helper()

	// The pool connects lazily, so no database is needed to construct it.
	pool, err := pgxpool.New(context.Background(), "postgres://user:password@localhost:5432/eats")
	if err != nil {
		t.Fatal(err)
	}

	e := commonHTTP.NewEcho()

	return Svc{echoRouter: e, dbPgx: pool, shutdownTimeout: shutdownTimeout}, e
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	return l.Addr().String()
}

func startSvc(t *testing.T, svc Svc, addr string) (cancel func(), runDone <-chan error) {
	t.Helper()

	ctx, cancelCtx := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx, addr) }()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/health")
		if err == nil {
			resp.Body.Close()
			return cancelCtx, done
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancelCtx()
	t.Fatal("the server did not start listening within 5s")
	return nil, nil
}

func requireRunReturns(t *testing.T, runDone <-chan error) error {
	t.Helper()

	select {
	case err := <-runDone:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return within 5s after the context was canceled")
		return nil
	}
}

func TestRun_StopsWhenContextIsCanceled(t *testing.T) {
	svc, _ := newTestSvc(t, 10*time.Second)
	cancel, runDone := startSvc(t, svc, freeAddr(t))

	cancel()

	if err := requireRunReturns(t, runDone); err != nil {
		t.Fatalf("Run should return nil after a clean shutdown, got: %v", err)
	}
}

func TestRun_WaitsForInFlightRequests(t *testing.T) {
	svc, e := newTestSvc(t, 10*time.Second)

	started := make(chan struct{})
	release := make(chan struct{})
	var handlerFinishedAt atomic.Int64
	e.GET("/slow", func(c echo.Context) error {
		close(started)
		<-release
		handlerFinishedAt.Store(time.Now().UnixNano())
		return c.String(http.StatusOK, "done")
	})

	addr := freeAddr(t)
	cancel, runDone := startSvc(t, svc, addr)

	type result struct {
		status int
		body   string
		err    error
	}
	respCh := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/slow")
		if err != nil {
			respCh <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		respCh <- result{status: resp.StatusCode, body: string(body)}
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the request did not reach the handler within 5s")
	}
	cancel()

	// The service is shutting down with one request still running.
	// Let it finish now and check Run waited for it.
	time.Sleep(100 * time.Millisecond)
	close(release)

	err := requireRunReturns(t, runDone)
	runReturnedAt := time.Now()
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}

	r := <-respCh
	if r.err != nil {
		t.Fatalf("the in-flight request failed: %v", r.err)
	}
	if r.status != http.StatusOK || r.body != "done" {
		t.Fatalf("the in-flight request got %d %q, want 200 \"done\"", r.status, r.body)
	}

	finishedAt := time.Unix(0, handlerFinishedAt.Load())
	if handlerFinishedAt.Load() == 0 || runReturnedAt.Before(finishedAt) {
		t.Fatal("Run returned before the in-flight request finished: the process would exit with the request still running")
	}
}

func TestRun_GivesUpAfterShutdownTimeout(t *testing.T) {
	svc, e := newTestSvc(t, 200*time.Millisecond)

	hang := make(chan struct{})
	t.Cleanup(func() { close(hang) })
	e.GET("/hang", func(c echo.Context) error {
		<-hang
		return c.NoContent(http.StatusOK)
	})

	addr := freeAddr(t)
	cancel, runDone := startSvc(t, svc, addr)

	go func() {
		resp, err := http.Get("http://" + addr + "/hang")
		if err == nil {
			resp.Body.Close()
		}
	}()
	time.Sleep(100 * time.Millisecond)

	cancel()

	err := requireRunReturns(t, runDone)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run should give up on a stuck request after the shutdown timeout and return context.DeadlineExceeded, got: %v", err)
	}
}

func TestRun_ReturnsStartError(t *testing.T) {
	svc, _ := newTestSvc(t, 10*time.Second)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan error, 1)
	go func() { runDone <- svc.Run(ctx, l.Addr().String()) }()

	select {
	case err := <-runDone:
		var opErr *net.OpError
		if !errors.As(err, &opErr) {
			t.Fatalf("Run should return the listen error when the port is taken, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal(fmt.Sprintf("Run did not return within 5s when the port %s was taken", l.Addr()))
	}
}
