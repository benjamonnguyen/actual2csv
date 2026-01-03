package main

import (
	"encoding/csv"
	"encoding/json"
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

// GetAccountsResponse represents the response from /accounts endpoint
type GetAccountsResponse []Account

// Account represents an account in Actual
type Account struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

// GetTransactionsResponse represents the response from /transactions endpoint
type GetTransactionsResponse struct {
	Data []Transaction `json:"data"`
}

// Transaction represents a transaction in Actual
type Transaction struct {
	ID       string  `json:"id"`
	Account  string  `json:"account"`
	Category *string `json:"category,omitempty"`
	Amount   int64   `json:"amount"`
	// Payee         string  `json:"payee"` // This is a UUID, not name
	Notes         *string `json:"notes"`
	Date          string  `json:"date"` // YYYY-MM-DD
	ImportedPayee *string `json:"imported_payee,omitempty"`
	// Cleared       bool    `json:"cleared"`
	// Tombstone     bool    `json:"tombstone"`
	// Additional fields that may be present but not used:
	// IsParent            bool     `json:"is_parent,omitempty"`
	// IsChild             bool     `json:"is_child,omitempty"`
	// ParentID            *string  `json:"parent_id,omitempty"`
	// ImportedID          *string  `json:"imported_id,omitempty"`
	Error *string `json:"error,omitempty"`
	// StartingBalanceFlag bool     `json:"starting_balance_flag,omitempty"`
	// TransferID          *string  `json:"transfer_id,omitempty"`
	// SortOrder           int64    `json:"sort_order,omitempty"`
	// Schedule            *string  `json:"schedule,omitempty"`
	// Subtransactions     []string `json:"subtransactions,omitempty"`
}

// TransactionRow represents a row in the CSV output
type TransactionRow struct {
	AccountName  string `csv:"account"`
	Date         string `csv:"date"`
	Amount       string `csv:"amount"`
	Payee        string `csv:"payee"`
	CategoryName string `csv:"category"`
	Notes        string `csv:"notes"`
	Error        string `csv:"error"`
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

	// Create HTTP client with auth
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get current month in YYYY-MM format
	currentMonth := time.Now().Local().Format("2006-01")
	startDate := currentMonth + "-01"
	endDate := currentMonth + "-31" // This works for all months due to Go's time parsing

	// TODO Create or trucate %currentMonth.csv as var file

	// Fetch accounts
	accounts, err := fetchAccounts(client, cfg)
	if err != nil {
		log.Fatalf("Failed to fetch accounts: %v", err)
	}

	log.Printf("Found %d accounts", len(accounts))

	// Collect all transactions from all accounts
	var totalTransactions int
	for _, account := range accounts {
		if account.Closed {
			log.Printf("Skipping closed account: %s", account.Name)
			continue
		}

		transactions, err := fetchTransactions(client, cfg, account.ID, startDate, endDate)
		if err != nil {
			log.Printf("Failed to fetch transactions for account %s: %v", account.Name, err)
			continue
		}

		if len(transactions) == 0 {
			log.Printf("No transactions for account: %s", account.Name)
			continue
		}

		// TODO addToCsv(file, account, transactions)
		totalTransactions += len(transactions)
		log.Printf("Fetched %d transactions for account %s", len(transactions), account.Name)
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

func fetchAccounts(client *http.Client, cfg Config) ([]Account, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts", cfg.ActualAPIURL, cfg.BudgetSyncID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-api-key", cfg.ActualAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accounts GetAccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return accounts, nil
}

func fetchTransactions(client *http.Client, cfg Config, accountID, startDate, endDate string) ([]Transaction, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts/%s/transactions", cfg.ActualAPIURL, cfg.BudgetSyncID, accountID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-api-key", cfg.ActualAPIKey)

	// Add query parameters
	q := req.URL.Query()
	q.Add("since_date", startDate)
	q.Add("until_date", endDate)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var transactionsResp GetTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&transactionsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return transactionsResp.Data, nil
}

func writeAllCSV(cfg Config, month string, transactions []TransactionWithAccount) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cfg.TransactionOutputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Create filename: {output_dir}/{month}.csv
	filename := fmt.Sprintf("%s.csv", month)
	filepath := filepath.Join(cfg.TransactionOutputDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header based on TransactionRow struct fields
	header := []string{"account", "date", "amount", "payee", "category", "notes", "error"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write transactions
	for _, txWithAccount := range transactions {
		row := convertToTransactionRow(txWithAccount)
		record := []string{
			row.AccountName,
			row.Date,
			row.Amount,
			row.Payee,
			row.CategoryName,
			row.Notes,
			row.Error,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("writing CSV record: %w", err)
		}
	}

	return nil
}

func convertToTransactionRow(account, transaction) TransactionRow {
	tx := txWithAccount.Transaction

	// Determine payee: use ImportedPayee if available, otherwise payee UUID
	payee := tx.Payee
	if tx.ImportedPayee != nil && *tx.ImportedPayee != "" {
		payee = *tx.ImportedPayee
	}

	// Determine notes
	notes := ""
	if tx.Notes != nil {
		notes = *tx.Notes
	}

	// Determine error message
	errorMsg := ""
	if tx.Error != nil && *tx.Error != "" {
		errorMsg = "[FIXME] " + *tx.Error
	}

	// Convert amount from cents to dollars with 2 decimal places
	amount := fmt.Sprintf("%.2f", float64(tx.Amount)/100.0)

	return TransactionRow{
		AccountName:  txWithAccount.AccountName,
		Date:         tx.Date,
		Amount:       amount,
		Payee:        payee,
		CategoryName: "FIXME", // Default as requested
		Notes:        notes,
		Error:        errorMsg,
	}
}
