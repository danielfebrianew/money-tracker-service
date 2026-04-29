package transactions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestSmartParserUsesKieProvider(t *testing.T) {
	var gotRequest kieResponseRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/v1/responses" {
			t.Fatalf("path = %s, want /codex/v1/responses", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %s, want bearer key", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{
					"type": "message",
					"content": [
						{
							"type": "output_text",
							"text": "{\"intent\":\"TRANSACTION\",\"jumlah\":25000,\"deskripsi\":\"Makan siang\",\"kategori\":\"Makan\",\"tipe\":\"OUT\",\"confidence\":0.92}"
						}
					]
				}
			],
			"status": "completed"
		}`))
	}))
	defer server.Close()

	parser := NewSmartParser(config.Config{
		OpenAIAPIKey:          "test-key",
		OpenAIModel:           "gpt-5-4",
		OpenAIBaseURL:         server.URL,
		OpenAIReasoningEffort: "high",
	}, nil)

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
	if parsed.Deskripsi != "Makan siang" {
		t.Fatalf("deskripsi = %s, want Makan siang", parsed.Deskripsi)
	}
	if gotRequest.Model != "gpt-5-4" {
		t.Fatalf("model = %s, want gpt-5-4", gotRequest.Model)
	}
	if gotRequest.Stream {
		t.Fatal("stream = true, want false")
	}
	if gotRequest.Reasoning == nil || gotRequest.Reasoning.Effort != "high" {
		t.Fatalf("reasoning = %+v, want high", gotRequest.Reasoning)
	}
	if len(gotRequest.Input) != 2 {
		t.Fatalf("input length = %d, want 2", len(gotRequest.Input))
	}
}

func TestSmartParserFallsBackWhenKieProviderFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	parser := NewSmartParser(config.Config{
		OpenAIAPIKey:  "test-key",
		OpenAIBaseURL: server.URL,
	}, nil)

	parsed, err := parser.ParseMessage(context.Background(), "makan siang 25rb")
	if err != nil {
		t.Fatalf("ParseMessage returned error: %v", err)
	}
	if parsed.Intent != "TRANSACTION" || parsed.Jumlah != 25000 || parsed.Kategori != "Makan" {
		t.Fatalf("fallback parsed = %+v, want local transaction parse", parsed)
	}
}
