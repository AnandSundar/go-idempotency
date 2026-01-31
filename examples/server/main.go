package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/go-idempotency"
	"github.com/yourusername/go-idempotency/store"
)

type PaymentRequest struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type PaymentResponse struct {
	ID        string    `json:"id"`
	Amount    int       `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Create store
	memStore := store.NewMemoryStore()

	// Create handler
	mux := http.NewServeMux()
	mux.HandleFunc("/api/payment", handlePayment)

	// Wrap with idempotency middleware
	handler := idempotency.Middleware(
		memStore,
		idempotency.WithTTL(24*time.Hour),
	)(mux)

	fmt.Println("Server starting on :8080")
	fmt.Println("\nTry:")
	fmt.Println(`  curl -X POST http://localhost:8080/api/payment \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -H "Idempotency-Key: payment-123" \`)
	fmt.Println(`    -d '{"amount":1000,"currency":"USD"}'`)
	fmt.Println("\nRun the same command again to see cached response!")

	log.Fatal(http.ListenAndServe(":8080", handler))
}

func handlePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Simulate payment processing
	time.Sleep(100 * time.Millisecond)

	response := PaymentResponse{
		ID:        fmt.Sprintf("pay_%d", time.Now().Unix()),
		Amount:    req.Amount,
		Currency:  req.Currency,
		Status:    "completed",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
