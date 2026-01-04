package main

import (
	"encoding/csv"
	"io"
	"strconv"
)

var headers = []string{
	"account",
	"date",
	"payee",
	"amount",
	"category",
	"notes",
}

type CSVWriter interface {
	Add(Account, []Transaction) error
}

type csvWriter struct {
	w           *csv.Writer
	categoryMap map[string]Category
	payeeMap    map[string]Payee
}

func NewCSVWriter(w io.Writer, categories map[string]Category, payeeMap map[string]Payee) CSVWriter {
	o := &csvWriter{
		w:           csv.NewWriter(w),
		categoryMap: categories,
		payeeMap:    payeeMap,
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
	var accountName, payeeName, categoryName string

	if p := w.payeeMap[transaction.PayeeID]; p != (Payee{}) {
		payeeName = p.Name
	}

	if c := w.categoryMap[transaction.CategoryID]; c != (Category{}) {
		if c.IsIncome {
			// flip posting source / destination
			transaction.Amount *= -1
			categoryName = account.Name
			accountName = c.Name
		} else {
			categoryName = c.Name
			accountName = account.Name
		}
	}

	amount := strconv.Itoa(transaction.Amount)

	return []string{
		accountName,
		transaction.Date,
		payeeName,
		amount,
		categoryName,
		transaction.Notes,
	}
}
