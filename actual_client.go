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

type ActualClient interface {
	FetchAccounts() (FetchAccountsResponse, error)
	FetchTransactions(accountID, startDate, endDate string) (FetchTransactionsResponse, error)
	/* @implement FetchCategories() (FetchCategoriesResponse, error)
	 path: /budgets/{budgetSyncId}/categories
	 payload example: {
		"data": [
			{
				"id": "106963b3-ab82-4734-ad70-1d7dc2a52ff4",
				"name": "For Spending",
			}
		]
	}
	*/

	/* @implement FetchPayees() (FetchPayeesResponse, error)
	 path: /budgets/{budgetSyncId}/payees
	 payload example: {
		"data": [
			{
				"id": "f733399d-4ccb-4758-b208-7422b27f650a",
				"name": "Fidelity",
			}
		]
	}
	*/
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
