package service

import (
	"context"
	"testing"

	"money-management-service/internal/config"
)

func TestSmartParserParsesExpense(t *testing.T) {
	parser := NewSmartParser(config.Config{}, nil)

	parsed, err := parser.ParseMessage(context.Background(), "makan siang 25rb")
	if err != nil {
		t.Fatalf("ParseMessage returned error: %v", err)
	}
	if parsed.Intent != "TRANSACTION" {
		t.Fatalf("intent = %s, want TRANSACTION", parsed.Intent)
	}
	if parsed.Jumlah != 25000 {
		t.Fatalf("jumlah = %d, want 25000", parsed.Jumlah)
	}
	if parsed.Kategori != "Makan" {
		t.Fatalf("kategori = %s, want Makan", parsed.Kategori)
	}
	if parsed.Tipe != "OUT" {
		t.Fatalf("tipe = %s, want OUT", parsed.Tipe)
	}
}

func TestSmartParserParsesGroupTransaction(t *testing.T) {
	parser := NewSmartParser(config.Config{}, nil)

	parsed, err := parser.ParseMessage(context.Background(), "grup keluarga makan 150rb")
	if err != nil {
		t.Fatalf("ParseMessage returned error: %v", err)
	}
	if parsed.Intent != "GROUP_TX" {
		t.Fatalf("intent = %s, want GROUP_TX", parsed.Intent)
	}
	if parsed.GroupName == nil || *parsed.GroupName != "keluarga" {
		t.Fatalf("group = %v, want keluarga", parsed.GroupName)
	}
	if parsed.Jumlah != 150000 {
		t.Fatalf("jumlah = %d, want 150000", parsed.Jumlah)
	}
}

func TestSmartParserParsesBalanceIntent(t *testing.T) {
	parser := NewSmartParser(config.Config{}, nil)

	parsed, err := parser.ParseMessage(context.Background(), "saldo")
	if err != nil {
		t.Fatalf("ParseMessage returned error: %v", err)
	}
	if parsed.Intent != "QUERY_BALANCE" {
		t.Fatalf("intent = %s, want QUERY_BALANCE", parsed.Intent)
	}
}
