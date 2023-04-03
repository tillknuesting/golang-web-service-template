package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// curl -X POST -H "Content-Type: application/json" -d '{"key": "key1", "value": "value1"}' http://localhost:8080/set
// curl -X POST -H "Content-Type: application/json" -d '{"key": "key1"}' http://localhost:8080/get

type Key string

type Value string

type SetRequest struct {
	Key   Key   `json:"key"`
	Value Value `json:"value"`
}

type GetRequest struct {
	Key Key `json:"key"`
}

type GetResponse struct {
	Value Value `json:"value"`
}

type ServerConfig struct {
	ServiceName             string
	ServerAddress           string
	ShutdownTimeout         time.Duration
	EnableLoggingMiddleware bool
	ServiceVersion          string
}

type KeyValueStore struct {
	sync.Mutex
	kvMap map[Key]Value
}

// go build -ldflags "-X main.version=1.5.0" -o main service.go
var version string //TODO: Consinder to not use global variable

func main() {
	// there is a hierarchy: provided flags, then environment variables, then default values
	var (
		serverPort      = flag.String("address", useEnvOrDefaultIfNotSet(os.Getenv("SERVER_ADDRESS"), "localhost:8080").(string), "server address")
		shutdownTimeout = flag.Duration("shutdown-timeout", useEnvOrDefaultIfNotSet(os.Getenv("SHUTDOWN_TIMEOUT"),
			time.Second*10).(time.Duration), "shutdown timeout e.g. 10s")
		enableLoggingMiddleware = flag.Bool("enable-logging-middleware", useEnvOrDefaultIfNotSet(os.Getenv("ENABLE_LOGGING_MIDDLEWARE"), false).(bool), "enable logging middleware")
	)

	flag.Parse()

	env := ServerConfig{
		ServiceName:             "key-value-service-v1",
		ServerAddress:           *serverPort,
		ShutdownTimeout:         *shutdownTimeout,
		EnableLoggingMiddleware: *enableLoggingMiddleware,
		ServiceVersion:          version,
	}

	log.Println(env)

	env.server()
}

// useEnvOrDefaultIfNotSet returns the value of the environment variable if it is set, otherwise it returns the default value
// this is useful for setting default values for flags but also allowing them to be overridden by environment variables
func useEnvOrDefaultIfNotSet(envValue interface{}, defaultValue interface{}) interface{} {
	if envValue == nil {
		return defaultValue
	}
	switch v := envValue.(type) {
	case string:
		if len(v) == 0 {
			return defaultValue
		}
	case time.Duration:
		if v == 0 {
			return defaultValue.(time.Duration)
		}
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
	return envValue
}

func (env *ServerConfig) server() {
	kvStore := KeyValueStore{
		kvMap: make(map[Key]Value),
	}

	endpoints := map[string]http.HandlerFunc{
		"/healthz": LivenessProbeHandler,
		"/readyz":  ReadinessProbeHandler,
		"/get":     kvStore.GetHandler,
		"/set":     kvStore.SetHandler,
	}

	handler := func(h http.HandlerFunc) http.HandlerFunc {
		if env.EnableLoggingMiddleware {
			return MiddlewareLogRequest(h)
		}
		return h
	}

	mux := http.NewServeMux()

	for path, ep := range endpoints {
		mux.HandleFunc(path, handler(ep))
	}

	// Create the server
	server := http.Server{
		Addr:         env.ServerAddress,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start the server
	go func() {
		log.Println("starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), env.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}

	log.Println("Server shut down successfully")
}

// LivenessProbeHandler handles the liveness probe
func LivenessProbeHandler(w http.ResponseWriter, r *http.Request) {
	// TDOO: Add more checks here, perhaps introduce global state to check if the server is still alive
	log.Println("Liveness probe called", r.URL.Path)
	w.WriteHeader(http.StatusOK)
}

// ReadinessProbeHandler handles the readiness probe
func ReadinessProbeHandler(w http.ResponseWriter, r *http.Request) {
	// TDOO: Add more checks here, perhaps introduce global state to check if the server ready to serve requests
	log.Println("Readiness probe called", r.URL.Path)
	w.WriteHeader(http.StatusOK)
}

// SetHandler handles the set request
func (kv *KeyValueStore) SetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	var payload SetRequest
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	kv.Lock()
	defer kv.Unlock()

	kv.kvMap[payload.Key] = payload.Value

	fmt.Fprintln(w, http.StatusAccepted)
}

// GetHandler returns the value for a given key
func (kv *KeyValueStore) GetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload GetRequest
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	kv.Lock()
	defer kv.Unlock()

	value, ok := kv.kvMap[payload.Key]
	if !ok {
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	response := GetResponse{Value: value}
	json.NewEncoder(w).Encode(response)
}

// MiddlewareLogRequest logs the request method and URL path
func MiddlewareLogRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log the request method and URL path
		log.Printf("Request: %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Log the request headers.
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("Header: %s=%s", name, value)
			}
		}

		// Call the next handler in the chain
		next(w, r)
	}
}
