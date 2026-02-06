package safego

import (
	"context"
	"log"
	"runtime/debug"
)

// Go runs a function in a goroutine with panic recovery.
// If the function panics, the panic is logged and the goroutine exits gracefully.
func Go(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] in %s: %v\n%s", name, r, debug.Stack())
			}
		}()
		fn()
	}()
}

// GoWithContext runs a function in a goroutine with panic recovery and context support.
// The function receives the context and can check for cancellation.
func GoWithContext(ctx context.Context, name string, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] in %s: %v\n%s", name, r, debug.Stack())
			}
		}()
		fn(ctx)
	}()
}

// Recover is a deferred function that recovers from panics and logs them.
// Use this at the start of any function that should not crash on panic.
// Example: defer safego.Recover("functionName")
func Recover(name string) {
	if r := recover(); r != nil {
		log.Printf("[PANIC RECOVERED] in %s: %v\n%s", name, r, debug.Stack())
	}
}

// RecoverWithCallback is a deferred function that recovers from panics,
// logs them, and calls a callback function. Useful for cleanup or retry logic.
func RecoverWithCallback(name string, callback func(panicValue interface{})) {
	if r := recover(); r != nil {
		log.Printf("[PANIC RECOVERED] in %s: %v\n%s", name, r, debug.Stack())
		if callback != nil {
			callback(r)
		}
	}
}
