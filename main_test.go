package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFrontendHandler(t *testing.T) {
	mockBackend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend Response"))
	}))
	defer mockBackend1.Close()

	config := Config{
		HealthCheckInterval: "5s",
		FrontendPort:        "9090",
		BackendURLs:         []string{mockBackend1.URL},
	}

	// Directly set the backends as healthy
	setBackendHealth(mockBackend1.URL, true)

	handler := frontendHandler(config)

	req, err := http.NewRequest("GET", "http://localhost:9090", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Manually set the RemoteAddr to simulate client IP
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Add("User-Agent", "TestAgent")
	req.Header.Add("Accept", "*/*")

	rr := httptest.NewRecorder()
	handler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expectedBody := "Backend Response"
	if rr.Body.String() != expectedBody {
		t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestFrontendHandlerNoHealthyBackend(t *testing.T) {
	config := Config{
		HealthCheckInterval: "5s",
		FrontendPort:        "9090",
		BackendURLs:         []string{"http://localhost:9091"}, // Non-existent backend
	}

	handler := frontendHandler(config)

	req, err := http.NewRequest("GET", "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Manually set the RemoteAddr to simulate client IP
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Add("User-Agent", "TestAgent")
	req.Header.Add("Accept", "*/*")

	rr := httptest.NewRecorder()
	handler(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusServiceUnavailable)
	}

	expectedBody := "No healthy backend available\n"
	if rr.Body.String() != expectedBody {
		t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expectedBody)
	}
}

func TestHealthCheck(t *testing.T) {
	mockBackend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockBackend1.Close()

	mockBackend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // Simulate unhealthy backend
	}))
	defer mockBackend2.Close()

	config := Config{
		HealthCheckInterval: "100ms",                                      // Shorter interval for testing purposes
		BackendURLs:         []string{mockBackend1.URL, mockBackend2.URL}, // Use mock server URLs
	}

	// Start health check
	go healthCheck(config)

	// Allow some time for health checks to run
	time.Sleep(300 * time.Millisecond)

	// Check backend health status
	if !healthyBackends[mockBackend1.URL] {
		t.Errorf("Backend %s should be healthy", mockBackend1.URL)
	}

	if healthyBackends[mockBackend2.URL] {
		t.Errorf("Backend %s should be unhealthy", mockBackend2.URL)
	}
}

func TestRoundRobinLoadBalancing(t *testing.T) {
	mockBackend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Backend 1")
	}))
	defer mockBackend1.Close()

	mockBackend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Backend 2")
	}))
	defer mockBackend2.Close()

	config := Config{
		HealthCheckInterval: "5s",
		FrontendPort:        "9090",
		BackendURLs:         []string{mockBackend1.URL, mockBackend2.URL},
	}

	// Directly set the backends as healthy
	setBackendHealth(mockBackend1.URL, true)
	setBackendHealth(mockBackend2.URL, true)

	fmt.Println()
	reqCount := 10 // Number of requests to send
	expectedBackend1Count := reqCount / 2
	expectedBackend2Count := reqCount / 2

	// Send requests to the frontend handler
	for i := 0; i < reqCount; i++ {
		req, err := http.NewRequest("GET", "http://localhost:9090", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Manually set the RemoteAddr to simulate client IP
		req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", i)
		req.Header.Add("User-Agent", "TestAgent")
		req.Header.Add("Accept", "*/*")

		rr := httptest.NewRecorder()

		handler := frontendHandler(config)
		handler(rr, req)

		fmt.Println(rr.Body.String())

		// Check which backend server handled the request
		if rr.Body.String() == "Backend 1\n" {
			expectedBackend1Count--
		} else if rr.Body.String() == "Backend 2\n" {
			expectedBackend2Count--
		} else {
			t.Errorf("Unexpected response from frontend handler: %s", rr.Body.String())
		}
	}

	// Verify that each backend server received approximately an equal number of requests
	if expectedBackend1Count != 0 || expectedBackend2Count != 0 {
		t.Errorf("Round-robin load balancing failed: Backend 1 count = %d, Backend 2 count = %d", expectedBackend1Count, expectedBackend2Count)
	}
}

func TestSetBackendHealth(t *testing.T) {
	backendURL := "http://localhost:9090"
	setBackendHealth(backendURL, true)

	if !healthyBackends[backendURL] {
		t.Errorf("Expected backend %s to be healthy", backendURL)
	}

	setBackendHealth(backendURL, false)

	if healthyBackends[backendURL] {
		t.Errorf("Expected backend %s to be unhealthy", backendURL)
	}
}
