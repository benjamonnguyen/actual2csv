package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type FetchAccountsResponse struct {
	Data []Account `json:"data"`
}

type Account struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

type FetchTransactionsResponse struct {
	Data []Transaction `json:"data"`
}

type Transaction struct {
	ID         string `json:"id"`
	AccountID  string `json:"account"`
	CategoryID string `json:"category"`
	Amount     int64  `json:"amount"`
	PayeeID    string `json:"payee"`
	Notes      string `json:"notes"`
	Date       string `json:"date"` // YYYY-MM-DD
	Error      string `json:"error"`
	// ImportedPayee *string `json:"imported_payee,omitempty"`
	// Cleared       bool    `json:"cleared"`
	// Tombstone     bool    `json:"tombstone"`
	// Additional fields that may be present but not used:
	// IsParent            bool     `json:"is_parent,omitempty"`
	// IsChild             bool     `json:"is_child,omitempty"`
	// ParentID            *string  `json:"parent_id,omitempty"`
	// ImportedID          *string  `json:"imported_id,omitempty"`
	// StartingBalanceFlag bool     `json:"starting_balance_flag,omitempty"`
	// TransferID          *string  `json:"transfer_id,omitempty"`
	// SortOrder           int64    `json:"sort_order,omitempty"`
	// Schedule            *string  `json:"schedule,omitempty"`
	// Subtransactions     []string `json:"subtransactions,omitempty"`
}

type FetchCategoriesResponse struct {
	Data []Category `json:"data"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FetchPayeesResponse struct {
	Data []Payee `json:"data"`
}

type Payee struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ActualClient interface {
	FetchAccounts() (FetchAccountsResponse, error)
	FetchTransactions(accountID, startDate, endDate string) (FetchTransactionsResponse, error)
	FetchCategories() (FetchCategoriesResponse, error)
	FetchPayees() (FetchPayeesResponse, error)
}

type actualClient struct {
	cfg    Config
	client *http.Client
}

func NewActualClient(cfg Config, client *http.Client) ActualClient {
	return &actualClient{
		cfg:    cfg,
		client: client,
	}
}

func (c *actualClient) FetchAccounts() (FetchAccountsResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts", c.cfg.ActualAPIURL, c.cfg.BudgetSyncID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return FetchAccountsResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-api-key", c.cfg.ActualAPIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return FetchAccountsResponse{}, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return FetchAccountsResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accounts FetchAccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return FetchAccountsResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return accounts, nil
}

func (c *actualClient) FetchTransactions(accountID, startDate, endDate string) (FetchTransactionsResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/accounts/%s/transactions", c.cfg.ActualAPIURL, c.cfg.BudgetSyncID, accountID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return FetchTransactionsResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-api-key", c.cfg.ActualAPIKey)

	// Add query parameters
	q := req.URL.Query()
	q.Add("since_date", startDate)
	q.Add("until_date", endDate)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return FetchTransactionsResponse{}, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return FetchTransactionsResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var transactionsResp FetchTransactionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&transactionsResp); err != nil {
		return FetchTransactionsResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return transactionsResp, nil
}

func (c *actualClient) FetchCategories() (FetchCategoriesResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/categories", c.cfg.ActualAPIURL, c.cfg.BudgetSyncID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return FetchCategoriesResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-api-key", c.cfg.ActualAPIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return FetchCategoriesResponse{}, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return FetchCategoriesResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var categoriesResp FetchCategoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&categoriesResp); err != nil {
		return FetchCategoriesResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return categoriesResp, nil
}

func (c *actualClient) FetchPayees() (FetchPayeesResponse, error) {
	url := fmt.Sprintf("%s/budgets/%s/payees", c.cfg.ActualAPIURL, c.cfg.BudgetSyncID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return FetchPayeesResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-api-key", c.cfg.ActualAPIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return FetchPayeesResponse{}, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return FetchPayeesResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var payeesResp FetchPayeesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payeesResp); err != nil {
		return FetchPayeesResponse{}, fmt.Errorf("decoding response: %w", err)
	}

	return payeesResp, nil
}
