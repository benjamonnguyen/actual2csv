package main

import (
	"encoding/csv"
	"fmt"
	"io"
)

var headers = []string{
	"account",
	"date",
	"amount",
	"payee",
	"category",
	"notes",
	"error",
}

type CSVWriter interface {
	Add(Account, []Transaction) error
}

type csvWriter struct {
	w *csv.Writer
}

func NewCSVWriter(w io.Writer) CSVWriter {
	o := &csvWriter{
		w: csv.NewWriter(w),
	}
	if err := o.w.Write(headers); err != nil {
		panic(err)
	}
	o.w.Flush()
	return o
}

func (w *csvWriter) Add(acct Account, txns []Transaction) error {
	if len(txns) == 0 {
		return nil
	}
	var rows [][]string
	for _, txn := range txns {
		row := w.transactionToRow(acct, txn)
		rows = append(rows, row)
	}
	if err := w.w.WriteAll(rows); err != nil {
		return err
	}
	w.w.Flush()
	return nil
}

func (w *csvWriter) transactionToRow(account Account, transaction Transaction) []string {
	payee := "FIXME"
	if transaction.ImportedPayee != nil && *transaction.ImportedPayee != "" {
		payee = *transaction.ImportedPayee
	}

	notes := ""
	if transaction.Notes != nil {
		notes = *transaction.Notes
	}

	errorMsg := ""
	if transaction.Error != nil && *transaction.Error != "" {
		errorMsg = "[FIXME] " + *transaction.Error
	}

	// Convert amount from cents to dollars with 2 decimal places
	amount := fmt.Sprintf("%.2f", float64(transaction.Amount)/100.0)

	category := "FIXME"
	if transaction.Category != nil && *transaction.Category != "" {
		category = *transaction.Category
	}

	return []string{
		account.Name,
		transaction.Date,
		amount,
		payee,
		category,
		notes,
		errorMsg,
	}
}
