package middleware

import (
	"testing"

	"time"

	"github.com/achilleasa/usrv"
)

func TestThrotleErrors(t *testing.T) {
	handler := func(req, res usrv.Message) {
	}

	didPanic := false

	catchPanicInThrottleMiddleware(0, time.Millisecond*1, handler, &didPanic)
	if !didPanic {
		t.Fatalf("Expected throttle middleware to panic")
	}
	catchPanicInThrottleMiddleware(-1, 0, handler, &didPanic)
	if !didPanic {
		t.Fatalf("Expected throttle middleware to panic")
	}

}

/*
func TestThrottleTimeout(t *testing.T) {

	done := make(chan usrv.ResponseWriter)
	trigger := make(chan struct{})

	ep := usrv.Endpoint{
		Name: "throttleTest",
		Handler: usrv.HandlerFunc(func(ctx context.Context, rw usrv.ResponseWriter, req *usrv.Message) {
			rw.Header().Set("status", "ok")

			// Block till we are triggered
			<-trigger
		}),
	}

	// Apply throttle (1 req, 1ms timeout)
	err := Throttle(1, time.Millisecond*1)(&ep)
	if err != nil {
		t.Fatalf("Throttle invocation failed with error: %s", err.Error())
	}

	// Spawn requests
	for i := 0; i < 2; i++ {
		go func() {
			w := usrvtest.NewRecorder()
			ep.Handler.Serve(context.Background(), w, nil)

			// Signal that we are done
			done <- w
		}()
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {

		// Wait 2 ms. The second request should have timed-out by now
		<-time.After(time.Millisecond * 2)

		var rw usrv.ResponseWriter
		rw = <-done

		errMsg := rw.Header().Get("error")
		if errMsg == nil {
			t.Fatalf("Expected request to fail")
		} else if errMsg.(string) != usrv.ErrTimeout.Error() {
			t.Fatalf("Expected request to fail with ErrTimeout. Instead failed with: %s", errMsg)
		}

		// Allow first request to finish
		trigger <- struct{}{}
		rw = <-done

		if rw.Header().Get("status") == nil || rw.Header().Get("error") != nil {
			t.Fatalf("Expected request to complete successfully")
		}

		wg.Done()
	}()

	wg.Wait()
}

func TestThrottleCancellation(t *testing.T) {

	done := make(chan usrv.ResponseWriter)
	trigger := make(chan struct{})
	ctx, cancelCtx := context.WithCancel(context.Background())

	ep := usrv.Endpoint{
		Name: "throttleTest",
		Handler: usrv.HandlerFunc(func(ctx context.Context, rw usrv.ResponseWriter, req *usrv.Message) {
			rw.Header().Set("status", "ok")

			// Block till we are triggered
			<-trigger
		}),
	}

	// Apply throttle (1 req, no timeout)
	err := Throttle(1, 0)(&ep)
	if err != nil {
		t.Fatalf("Throttle invocation failed with error: %s", err.Error())
	}

	// Spawn requests
	for i := 0; i < 2; i++ {
		go func() {
			w := usrvtest.NewRecorder()
			ep.Handler.Serve(ctx, w, nil)

			// Signal that we are done
			done <- w
		}()
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// Cancel context
		cancelCtx()
		var rw usrv.ResponseWriter
		rw = <-done

		errMsg := rw.Header().Get("error")
		if errMsg == nil {
			t.Fatalf("Expected request to fail")
		} else if errMsg.(string) != usrv.ErrCancelled.Error() {
			t.Fatalf("Expected request to fail with ErrCancelled. Instead failed with: %s", errMsg)
		}

		// Allow first request to finish
		trigger <- struct{}{}
		rw = <-done

		if rw.Header().Get("status") == nil || rw.Header().Get("error") != nil {
			t.Fatalf("Expected request to complete successfully")
		}

		wg.Done()
	}()

	wg.Wait()
}
*/

func catchPanicInThrottleMiddleware(maxConcurrent int, timeout time.Duration, handler usrv.Handler, didPanic *bool) {
	*didPanic = false
	defer func() {
		err := recover()
		if err != nil {
			*didPanic = true
		}
	}()

	Throttle(maxConcurrent, timeout, handler)
}
