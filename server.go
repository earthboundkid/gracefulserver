package gracefulserver

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Serve starts an HTTP listener on the port specified by environmental
// variable PORT (8080 if not set). Requests
// will be logged by the Logger middleware. Serve blocks until SIGINT or
// SIGTERM is received and the listener is closed.
func Serve(handler http.Handler) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{Addr: ":" + port, Handler: Logger(handler)}

	errc := make(chan error)
	go func() {
		log.Printf("Begin listening on port %s", port)
		// service connections
		errc <- srv.ListenAndServe()
	}()

	<-stopChan // wait for system signal
	log.Println("Shutting down server...")

	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, c := context.WithTimeout(context.Background(), Timeout)
	defer c()
	srv.Shutdown(ctx)

	select {
	case err := <-errc:
		log.Printf("Finished listening: %v\n", err)
	case <-ctx.Done():
		log.Println("Graceful shutdown timed out")
	}

	log.Println("Server stopped")
}

// Logger is the logging middleware for gracefulserver. By default it logs the
// URL, UserAgent, and duration of requests with Go standard logger.
var Logger = func(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("Served %s for %q in %v", r.URL, r.UserAgent(), time.Since(start))
	})

}

var (
	// Timeout is the amount of time the server will wait for requests to finish during shutdown
	Timeout = 5 * time.Second
)
