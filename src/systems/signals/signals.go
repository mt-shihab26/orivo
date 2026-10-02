package signals

import (
	"os"
	"os/signal"
	"syscall"
)

func OnInterrupt(fn func()) {
	received := make(chan os.Signal, 1)
	signal.Notify(received, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-received
		fn()
	}()
}
