package main

import (
	"flag"
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
	// Parse command line flags
	var fromFlag, toFlag, cfgFlag string
	flag.StringVar(&fromFlag, "from", "", "Start month in YYYY-MM format")
	flag.StringVar(&toFlag, "to", "", "End month in YYYY-MM format")
	flag.StringVar(&cfgFlag, "cfg", "./.env", "Path to configuration file")
	flag.Parse()

	// Validate from/to flags: either both present or neither
	if (fromFlag == "") != (toFlag == "") {
		log.Fatal("Both -from and -to must be specified together, or neither")
	}

	// Load environment variables
	if err := godotenv.Load(cfgFlag); err != nil {
		log.Printf("Warning: Error loading configuration file: %v", err)
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

	// Determine date range based on flags
	var startDate, endDate, monthRange string
	if fromFlag == "" && toFlag == "" {
		// Use current month
		currentMonth := time.Now().Local().Format("2006-01")
		startDate = currentMonth + "-01"
		endDate = currentMonth + "-31" // This works for all months due to Go's time parsing
		monthRange = currentMonth
	} else {
		// Validate month formats
		fromTime, err := time.Parse("2006-01", fromFlag)
		if err != nil {
			log.Fatalf("Invalid -from format: %v", err)
		}
		toTime, err := time.Parse("2006-01", toFlag)
		if err != nil {
			log.Fatalf("Invalid -to format: %v", err)
		}
		// Validate that from is before to
		if !fromTime.Before(toTime) {
			log.Fatalf("-from must be before -to")
		}
		startDate = fromFlag + "-01"
		endDate = toFlag + "-31"
		monthRange = fmt.Sprintf("%s-%s", fromFlag, toFlag)
	}

	// Create file
	if err := os.MkdirAll(cfg.TransactionOutputDir, 0o755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	filename := fmt.Sprintf("%s.csv", monthRange)
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

	// Build name maps
	categoriesResp, err := actualClient.FetchCategories()
	if err != nil {
		failWithMsg(file, fmt.Sprintf("Failed to fetch categories: %s", err))
	}
	categoryMap := make(map[string]Category)
	for _, category := range categoriesResp.Data {
		categoryMap[category.ID] = category
	}

	payeesResp, err := actualClient.FetchPayees()
	if err != nil {
		failWithMsg(file, fmt.Sprintf("Failed to fetch payees: %s", err))
	}
	payeeMap := make(map[string]Payee)
	for _, payee := range payeesResp.Data {
		payeeMap[payee.ID] = payee
	}

	// Fetch accounts
	accountsResp, err := actualClient.FetchAccounts()
	if err != nil {
		failWithMsg(file, fmt.Sprintf("Failed to fetch accounts: %s", err))
	}
	accounts := accountsResp.Data
	log.Printf("Found %d accounts", len(accounts))

	// Write txns
	csvWriter := NewCSVWriter(file, categoryMap, payeeMap)
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
		log.Printf("Added %d transactions for account %s (%s)", len(transactions), account.Name, account.ID)
	}

	if totalTransactions == 0 {
		log.Println("No transactions found for any account")
		return
	}

	log.Printf("Written %d total transactions to CSV for range %s", totalTransactions, monthRange)
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
