package main

import (
	"encoding/csv"
	"fmt"
	"io"
)

var headers = []string{
	"account",
	"date",
	"payee",
	"amount",
	"category",
	"notes",
	"error",
}

type CSVWriter interface {
	Add(Account, []Transaction) error
}

type csvWriter struct {
	w             *csv.Writer
	categoryNames map[string]string
	payeeNames    map[string]string
}

func NewCSVWriter(w io.Writer, categoryNames, payeeNames map[string]string) CSVWriter {
	o := &csvWriter{
		w:             csv.NewWriter(w),
		categoryNames: categoryNames,
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
	if transaction.PayeeID != "" {
		if p := w.payeeNames[transaction.PayeeID]; p != "" {
			payee = p
		}
	}

	category := "FIXME"
	if transaction.CategoryID != "" {
		if c := w.categoryNames[transaction.CategoryID]; c != "" {
			category = c
		}
	}

	notes := transaction.Notes

	errorMsg := ""
	if transaction.Error != "" {
		errorMsg = "[FIXME] " + transaction.Error
	}

	// Convert amount from cents to dollars with 2 decimal places
	amount := fmt.Sprintf("%.2f", float64(transaction.Amount)/100.0)

	return []string{
		account.Name,
		transaction.Date,
		payee,
		amount,
		category,
		notes,
		errorMsg,
	}
}
