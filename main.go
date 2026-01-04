package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// Config holds environment configuration
type Config struct {
	BudgetSyncID         string
	ActualAPIKey         string
	ActualAPIURL         string
	TransactionOutputDir string
}

func main() {
	// Load environment variables
	if err := godotenv.Load("./.env"); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	cfg := Config{
		BudgetSyncID:         getEnv("BUDGET_SYNC_ID", ""),
		ActualAPIKey:         getEnv("ACTUAL_API_KEY", ""),
		ActualAPIURL:         getEnv("ACTUAL_API_URL", ""),
		TransactionOutputDir: getEnv("TRANSACTION_OUTPUT_DIR", ""),
	}

	// Validate config
	if cfg.BudgetSyncID == "" || cfg.ActualAPIKey == "" || cfg.ActualAPIURL == "" {
		log.Fatal("Missing required environment variables: BUDGET_SYNC_ID, ACTUAL_API_KEY, ACTUAL_API_URL")
	}

	// Get current month in YYYY-MM format
	currentMonth := time.Now().Local().Format("2006-01")
	startDate := currentMonth + "-01"
	endDate := currentMonth + "-31" // This works for all months due to Go's time parsing

	// Create file
	if err := os.MkdirAll(cfg.TransactionOutputDir, 0o755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	filename := fmt.Sprintf("%s.csv", currentMonth)
	filepath := filepath.Join(cfg.TransactionOutputDir, filename)
	file, err := os.Create(filepath)
	if err != nil {
		log.Fatalf("Failed to create CSV file: %v", err)
	}
	defer file.Close() //nolint

	// Client
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	actualClient := NewActualClient(cfg, client)

	// Build Categories map
	categoriesResp, err := actualClient.FetchCategories()
	if err != nil {
		failWithMsg(file, fmt.Sprintf("Failed to fetch categories: %s", err))
	}
	categoryNames := make(map[string]string)
	for _, category := range categoriesResp.Data {
		categoryNames[category.ID] = category.Name
	}

	// Fetch accounts
	accountsResp, err := actualClient.FetchAccounts()
	if err != nil {
		failWithMsg(file, fmt.Sprintf("Failed to fetch accounts: %s", err))
	}
	accounts := accountsResp.Data
	log.Printf("Found %d accounts", len(accounts))

	// Write txns
	csvWriter := NewCSVWriter(file, categoryNames)
	var totalTransactions int
	for _, account := range accounts {
		if account.Closed {
			log.Printf("Skipping closed account: %s", account.Name)
			continue
		}

		txnResponse, err := actualClient.FetchTransactions(account.ID, startDate, endDate)
		if err != nil {
			failWithMsg(file, fmt.Sprintf("Failed to fetch transactions for account %s: %v", account.Name, err))
			continue
		}
		transactions := txnResponse.Data

		if len(transactions) == 0 {
			log.Printf("No transactions for account: %s", account.Name)
			continue
		}

		if err := csvWriter.Add(account, transactions); err != nil {
			failWithMsg(file, fmt.Sprintf("Failed to write transactions for account %s: %v", account.Name, err))
		}
		totalTransactions += len(transactions)
		log.Printf("Added %d transactions for account %s", len(transactions), account.Name)
	}

	if totalTransactions == 0 {
		log.Println("No transactions found for any account")
		return
	}

	log.Printf("Written %d total transactions to CSV for month %s", totalTransactions, currentMonth)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func failWithMsg(w *os.File, msg string) {
	w.WriteString("[FIXME] " + msg) //nolint
	log.Fatal(msg)
}
