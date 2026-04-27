package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	"money-management-service/internal/model"
)

type OpenAIService interface {
	ParseMessage(ctx context.Context, message string) (*model.ParsedTransaction, error)
	ClassifyIntent(ctx context.Context, message string) (string, error)
}

type SmartParser struct {
	cfg   config.Config
	cache *cache.Cache
}

func NewSmartParser(cfg config.Config, cache *cache.Cache) *SmartParser {
	return &SmartParser{cfg: cfg, cache: cache}
}

func (s *SmartParser) ClassifyIntent(ctx context.Context, message string) (string, error) {
	parsed, err := s.ParseMessage(ctx, message)
	if err != nil {
		return "UNKNOWN", err
	}
	return parsed.Intent, nil
}

func (s *SmartParser) ParseMessage(ctx context.Context, message string) (*model.ParsedTransaction, error) {
	clean := strings.ToLower(strings.TrimSpace(message))
	if clean == "" {
		return &model.ParsedTransaction{Intent: "UNKNOWN", Confidence: 0.2}, nil
	}
	if containsAny(clean, "help", "bantuan", "cara pakai", "format") {
		return &model.ParsedTransaction{Intent: "HELP", Confidence: 1}, nil
	}
	if containsAny(clean, "saldo", "balance") {
		return &model.ParsedTransaction{Intent: "QUERY_BALANCE", Confidence: 1}, nil
	}
	if containsAny(clean, "laporan", "report", "rekap", "pengeluaran bulan", "pengeluaran minggu") {
		return &model.ParsedTransaction{Intent: "QUERY_REPORT", Confidence: 0.95}, nil
	}
	if containsAny(clean, "hai", "halo", "hello") && len(clean) <= 12 {
		return &model.ParsedTransaction{Intent: "CHITCHAT", Confidence: 0.9}, nil
	}

	amount, amountText := parseAmount(clean)
	if amount <= 0 {
		return &model.ParsedTransaction{Intent: "UNKNOWN", Confidence: 0.35}, nil
	}

	desc := cleanupDescription(clean, amountText)
	groupName := parseGroupName(clean)
	intent := "TRANSACTION"
	if groupName != nil {
		intent = "GROUP_TX"
	}
	tipe := "OUT"
	if containsAny(clean, "gaji", "bonus", "income", "pemasukan", "masuk", "terima", "dibayar") {
		tipe = "IN"
	}
	return &model.ParsedTransaction{
		Intent:     intent,
		Jumlah:     amount,
		Deskripsi:  title(desc),
		Kategori:   classifyCategory(clean),
		Tipe:       tipe,
		GroupName:  groupName,
		Confidence: 0.86,
	}, nil
}

func parseAmount(message string) (int, string) {
	re := regexp.MustCompile(`(?i)(\d+(?:[.,]\d{3})*(?:[.,]\d+)?|\d+)\s*(rb|ribu|k|jt|juta|mio)?`)
	matches := re.FindAllStringSubmatch(message, -1)
	for _, match := range matches {
		raw := match[1]
		unit := strings.ToLower(match[2])
		normalized := strings.ReplaceAll(raw, ".", "")
		if unit == "jt" || unit == "juta" || unit == "mio" {
			normalized = strings.ReplaceAll(raw, ",", ".")
			value, err := strconv.ParseFloat(normalized, 64)
			if err == nil {
				return int(value * 1000000), match[0]
			}
		}
		normalized = strings.ReplaceAll(normalized, ",", "")
		value, err := strconv.Atoi(normalized)
		if err != nil || value <= 0 {
			continue
		}
		if unit == "rb" || unit == "ribu" || unit == "k" {
			value *= 1000
		}
		if value >= 1000 {
			return value, match[0]
		}
	}
	return 0, ""
}

func cleanupDescription(message, amountText string) string {
	desc := strings.TrimSpace(strings.Replace(message, amountText, "", 1))
	desc = regexp.MustCompile(`(?i)\b(grup|group)\s+[a-z0-9_-]+`).ReplaceAllString(desc, "")
	desc = strings.TrimSpace(strings.Trim(desc, "-:,.; "))
	if desc == "" {
		return "Transaksi"
	}
	return desc
}

func parseGroupName(message string) *string {
	re := regexp.MustCompile(`(?i)\b(?:grup|group)\s+([a-z0-9_-]+)`)
	match := re.FindStringSubmatch(message)
	if len(match) < 2 {
		return nil
	}
	name := strings.TrimSpace(match[1])
	return &name
}

func classifyCategory(message string) string {
	switch {
	case containsAny(message, "makan", "kopi", "sarapan", "siang", "malam", "resto", "gofood", "grabfood"):
		return "Makan"
	case containsAny(message, "gojek", "grab", "bensin", "parkir", "tol", "transport", "ojek", "taxi"):
		return "Transport"
	case containsAny(message, "listrik", "air", "internet", "tagihan", "pulsa", "wifi"):
		return "Tagihan"
	case containsAny(message, "belanja", "market", "mall", "tokopedia", "shopee", "baju"):
		return "Belanja"
	case containsAny(message, "gaji", "bonus", "income", "pemasukan"):
		return "Pemasukan"
	default:
		return "Lainnya"
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func title(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
