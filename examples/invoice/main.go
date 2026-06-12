package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/gompdf/gompdf"
)

type LineItem struct {
	Description string
	Qty         int
	UnitPrice   float64
}

type InvoiceData struct {
	InvoiceNo   string
	IssueDate   string
	DueDate     string
	BillToName  string
	BillToLine1 string
	BillToLine2 string
	Items       []LineItem
	Subtotal    float64
	TaxRate     float64
	TaxAmount   float64
	Total       float64
}

func sampleData() InvoiceData {
	items := []LineItem{
		{"Consulting services", 12, 150.00},
		{"Design review", 4, 120.00},
		{"Support & maintenance", 6, 90.00},
	}
	var subtotal float64
	for _, it := range items {
		subtotal += float64(it.Qty) * it.UnitPrice
	}
	rate := 0.07
	return InvoiceData{
		InvoiceNo:   "INV-2025-091",
		IssueDate:   "2025-09-12",
		DueDate:     "2025-09-26",
		BillToName:  "Acme, Inc.",
		BillToLine1: "123 Market Street",
		BillToLine2: "San Francisco, CA 94107",
		Items:       items,
		Subtotal:    subtotal,
		TaxRate:     rate,
		TaxAmount:   subtotal * rate,
		Total:       subtotal * (1 + rate),
	}
}

func main() {
	data := sampleData()

	funcMap := template.FuncMap{
		"mul": func(a, b interface{}) float64 {
			toF := func(v interface{}) float64 {
				switch n := v.(type) {
				case int:
					return float64(n)
				case int64:
					return float64(n)
				case float64:
					return n
				case float32:
					return float64(n)
				default:
					return 0
				}
			}
			return toF(a) * toF(b)
		},
	}

	tmpl, err := template.New("invoice.tmpl.html").Funcs(funcMap).ParseFiles("invoice.tmpl.html")
	if err != nil {
		log.Fatalf("parse template: %v", err)
	}

	htmlPath := "invoice.html"
	f, err := os.Create(htmlPath)
	if err != nil {
		log.Fatalf("create html: %v", err)
	}
	if err := tmpl.Execute(f, data); err != nil {
		log.Fatalf("execute template: %v", err)
	}
	_ = f.Close()

	abs, _ := filepath.Abs(htmlPath)

	opts := gompdf.DefaultOptions()
	opts.MarginTop, opts.MarginRight, opts.MarginBottom, opts.MarginLeft = 36, 36, 36, 36
	opts.RenderBackgrounds = true
	opts.RenderBorders = true
	conv := gompdf.NewWithOptions(opts)

	out := "invoice.pdf"
	if err := conv.ConvertFile(abs, out); err != nil {
		log.Fatalf("convert: %v", err)
	}
	fmt.Printf("Wrote %s\n", out)
}
