package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Config struct {
	HealthCheckInterval string   `json:"health_check_interval"`
	FrontendPort        string   `json:"frontend_port"`
	BackendURLs         []string `json:"backend_urls"`
}

var (
	healthCheckInterval time.Duration           // Health check interval
	healthyBackends     = make(map[string]bool) // Map to track backend server health
	mu                  sync.Mutex              // Mutex to protect backend selection
	backendIndex        int                     // Index to track the next backend to which the request should be forwarded
)

// frontendHandler handles incoming requests, forwards them to the backend, and returns the combined response.
func frontendHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		method := r.Method
		url := r.URL.RequestURI() // Use RequestURI to get the path and query string
		protocol := r.Proto
		host := r.Host
		userAgent := r.Header.Get("User-Agent")
		accept := r.Header.Get("Accept")

		requestDetails := fmt.Sprintf("Received request from %s\n%s %s %s\nHost: %s\nUser-Agent: %s\nAccept: %s\n",
			clientIP, method, url, protocol, host, userAgent, accept)

		fmt.Println(requestDetails) // Log the request details

		// Forward the request to a healthy backend server using round-robin
		backendBaseURL := getNextHealthyBackendURL(config)
		if backendBaseURL == "" {
			http.Error(w, "No healthy backend available", http.StatusServiceUnavailable)
			return
		}

		backendURL := backendBaseURL + url
		backendReq, err := http.NewRequest(method, backendURL, r.Body)
		if err != nil {
			http.Error(w, "Error creating request to backend", http.StatusInternalServerError)
			return
		}

		// Copy headers
		for name, values := range r.Header {
			for _, value := range values {
				backendReq.Header.Add(name, value)
			}
		}

		client := &http.Client{}
		backendResp, err := client.Do(backendReq)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error forwarding request to backend", http.StatusInternalServerError)
			return
		}
		defer backendResp.Body.Close()

		// Copy backend response headers and status code
		for name, values := range backendResp.Header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(backendResp.StatusCode)

		// Log the response status from the backend
		fmt.Printf("Response from server: %s\n\n", backendResp.Status)

		// Read the backend response body
		body, err := io.ReadAll(backendResp.Body)
		if err != nil {
			http.Error(w, "Error reading response from backend", http.StatusInternalServerError)
			return
		}

		// Print the backend response body
		fmt.Println(string(body))

		// Write the backend response body to the client
		if _, err := io.Copy(w, bytes.NewReader(body)); err != nil {
			http.Error(w, "Error writing response to client", http.StatusInternalServerError)
			return
		}
	}
}

// healthCheck periodically checks the health of backend servers.
func healthCheck(config Config) {
	for {
		for _, backendURL := range config.BackendURLs {
			resp, err := http.Get(backendURL)
			if err != nil || resp.StatusCode != http.StatusOK {
				setBackendHealth(backendURL, false)
			} else {
				setBackendHealth(backendURL, true)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
		time.Sleep(healthCheckInterval)
	}
}

// getNextHealthyBackendURL returns the next healthy backend URL in a round-robin manner.
func getNextHealthyBackendURL(config Config) string {
	mu.Lock()
	defer mu.Unlock()

	// Ensure we always start checking from the current backendIndex
	startIndex := backendIndex

	for {
		// Increment backendIndex and wrap around if necessary
		backendIndex = (backendIndex + 1) % len(config.BackendURLs)

		// Check if the backend at the current index is healthy
		if healthy, ok := healthyBackends[config.BackendURLs[backendIndex]]; ok && healthy {
			return config.BackendURLs[backendIndex]
		}

		// If we have checked all backends and none are healthy, return an empty string
		if backendIndex == startIndex {
			return ""
		}
	}
}

// setBackendHealth sets the health status of a backend server.
func setBackendHealth(backendURL string, healthy bool) {
	mu.Lock()
	defer mu.Unlock()
	healthyBackends[backendURL] = healthy
}

// StartServer starts the load balancer server
func StartServer(config Config) {
	// Parse the health check interval
	var err error
	healthCheckInterval, err = time.ParseDuration(config.HealthCheckInterval)
	if err != nil {
		log.Fatalf("Invalid health check interval: %s\n", err)
	}

	// Start the backend health checker
	go healthCheck(config)

	// Start the frontend server
	port := config.FrontendPort
	frontendMux := http.NewServeMux()
	frontendMux.HandleFunc("/", frontendHandler(config))
	fmt.Printf("Starting frontend server on port %s\n", port)
	err = http.ListenAndServe(":"+port, frontendMux)
	if err != nil {
		log.Fatalf("Could not start frontend server: %s\n", err)
	}
}

func main() {
	// Load configuration
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Error opening config file: %s\n", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Error parsing config file: %s\n", err)
	}

	StartServer(config)
}
