package cleanup

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	// functions holds the cleanup functions to execute on program exit.
	functions []func()
	// lockFunctions is the lock for accessing the cleanup functions.
	lockFunctions sync.Mutex
)

// Register adds a function for cleanup.
// REMARK: The function may be called from a different Goroutine so the scope of the function needs to be accessed in a thread-safe manner. Calls to "Register" before "Init" are ineffective.
func Register(f func()) {
	lockFunctions.Lock()
	defer lockFunctions.Unlock()

	functions = append(functions, f)
}

var (
	// signalChannel holds the channel for notifying os signals.
	signalChannel chan os.Signal
	// handler synchronizes the cleanup handler.
	handler sync.WaitGroup
)

// Init sets up cleanup.
// If already initialized, subsequent calls to "Init" block until cleanup was triggered.
func Init() {
	handler.Wait()

	lockFunctions.Lock()
	defer lockFunctions.Unlock()

	functions = nil

	if signalChannel == nil { // Assume that we already set everything up.
		signalChannel = make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	}

	handler.Add(1)
	go func() {
		defer handler.Done()
		<-signalChannel

		lockFunctions.Lock()
		defer lockFunctions.Unlock()

		if len(functions) > 0 {
			fmt.Println("Graceful shutdown. Cleaning up...")
		}
		for _, f := range functions {
			f()
		}
		functions = nil
	}()
}

// Trigger executes the cleanup manually.
func Trigger() {
	if signalChannel == nil {
		panic("cleanup was never initialized")
	}

	signalChannel <- os.Interrupt // We react to all signals coming through the channel, so any would do.
	handler.Wait()
}
