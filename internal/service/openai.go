package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	"money-management-service/internal/model"
)

const (
	defaultKieBaseURL         = "https://api.kie.ai"
	defaultKieModel           = "gpt-5-4"
	defaultKieReasoningEffort = "low"
)

const smartParserInstructions = `You are a strict Indonesian personal-finance message parser.
Return only one valid JSON object, without markdown or extra text.

Schema:
{
  "intent": "TRANSACTION|GROUP_TX|QUERY_BALANCE|QUERY_REPORT|HELP|CHITCHAT|UNKNOWN",
  "jumlah": 0,
  "deskripsi": "",
  "kategori": "",
  "tipe": "IN|OUT",
  "group_name": null,
  "confidence": 0.0,
  "error": ""
}

Rules:
- jumlah must be an integer amount in Indonesian Rupiah.
- Parse rb/ribu/k as thousands and jt/juta/mio as millions.
- Use intent GROUP_TX only when the message includes a group name such as "grup keluarga" or "group kantor".
- Use tipe IN for income such as gaji, bonus, pemasukan, terima, dibayar; otherwise use OUT for expenses.
- Prefer kategori: Makan, Transport, Tagihan, Belanja, Pemasukan, Lainnya.
- For balance questions use QUERY_BALANCE, report/rekap questions use QUERY_REPORT, help/how-to questions use HELP, short greetings use CHITCHAT.
- For unknown messages return UNKNOWN with a short Indonesian error message.
- Use confidence from 0 to 1. Only use 0.8 or higher when the parsed transaction is clear.`

type OpenAIService interface {
	ParseMessage(ctx context.Context, message string) (*model.ParsedTransaction, error)
	ClassifyIntent(ctx context.Context, message string) (string, error)
}

type SmartParser struct {
	cfg        config.Config
	cache      *cache.Cache
	httpClient *http.Client
}

func NewSmartParser(cfg config.Config, cache *cache.Cache) *SmartParser {
	return &SmartParser{
		cfg:   cfg,
		cache: cache,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *SmartParser) ClassifyIntent(ctx context.Context, message string) (string, error) {
	parsed, err := s.ParseMessage(ctx, message)
	if err != nil {
		return "UNKNOWN", err
	}
	return parsed.Intent, nil
}

func (s *SmartParser) ParseMessage(ctx context.Context, message string) (*model.ParsedTransaction, error) {
	if s.providerEnabled() {
		parsed, err := s.parseWithKie(ctx, message)
		if err == nil && parsed != nil {
			return parsed, nil
		}
	}
	return parseMessageLocally(message), nil
}

func (s *SmartParser) providerEnabled() bool {
	return strings.TrimSpace(s.cfg.OpenAIAPIKey) != ""
}

func (s *SmartParser) parseWithKie(ctx context.Context, message string) (*model.ParsedTransaction, error) {
	body, err := json.Marshal(kieResponseRequest{
		Model:  s.openAIModel(),
		Stream: false,
		Input: []kieInputMessage{
			{
				Role: "developer",
				Content: []kieInputContent{
					{Type: "input_text", Text: smartParserInstructions},
				},
			},
			{
				Role: "user",
				Content: []kieInputContent{
					{Type: "input_text", Text: message},
				},
			},
		},
		Reasoning: &kieReasoning{Effort: s.reasoningEffort()},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.responsesURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.cfg.OpenAIAPIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("kie responses api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	outputText, err := extractKieOutputText(responseBody)
	if err != nil {
		return nil, err
	}
	return parseProviderTransaction(outputText)
}

func (s *SmartParser) openAIModel() string {
	modelName := strings.TrimSpace(s.cfg.OpenAIModel)
	if modelName == "" {
		return defaultKieModel
	}
	return modelName
}

func (s *SmartParser) reasoningEffort() string {
	switch strings.ToLower(strings.TrimSpace(s.cfg.OpenAIReasoningEffort)) {
	case "medium", "high", "xhigh":
		return strings.ToLower(strings.TrimSpace(s.cfg.OpenAIReasoningEffort))
	default:
		return defaultKieReasoningEffort
	}
}

func (s *SmartParser) responsesURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(s.cfg.OpenAIBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultKieBaseURL
	}
	if strings.HasSuffix(baseURL, "/responses") {
		return baseURL
	}
	if strings.HasSuffix(baseURL, "/codex/v1") {
		return baseURL + "/responses"
	}
	return baseURL + "/codex/v1/responses"
}

type kieResponseRequest struct {
	Model     string            `json:"model"`
	Stream    bool              `json:"stream"`
	Input     []kieInputMessage `json:"input"`
	Reasoning *kieReasoning     `json:"reasoning,omitempty"`
}

type kieInputMessage struct {
	Role    string            `json:"role"`
	Content []kieInputContent `json:"content"`
}

type kieInputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type kieReasoning struct {
	Effort string `json:"effort"`
}

type kieResponse struct {
	Output []kieOutputItem `json:"output"`
	Error  *kieError       `json:"error,omitempty"`
}

type kieError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type kieOutputItem struct {
	Type    string             `json:"type"`
	Content []kieOutputContent `json:"content"`
}

type kieOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func extractKieOutputText(body []byte) (string, error) {
	var response kieResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return extractKieSSEText(body)
	}
	if response.Error != nil {
		return "", fmt.Errorf("kie responses api error: %s", response.Error.Message)
	}

	var parts []string
	for _, output := range response.Output {
		if output.Type != "message" {
			continue
		}
		for _, content := range output.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				parts = append(parts, content.Text)
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("kie responses api returned no output text")
	}
	return strings.Join(parts, ""), nil
}

func extractKieSSEText(body []byte) (string, error) {
	var output strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event struct {
			Delta string `json:"delta"`
			Type  string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Type == "response.output_text.delta" {
			output.WriteString(event.Delta)
		}
	}
	if output.Len() == 0 {
		return "", fmt.Errorf("kie responses api returned unreadable response")
	}
	return output.String(), nil
}

func parseProviderTransaction(outputText string) (*model.ParsedTransaction, error) {
	jsonText, err := extractJSONObject(outputText)
	if err != nil {
		return nil, err
	}
	var parsed model.ParsedTransaction
	if err := json.Unmarshal([]byte(jsonText), &parsed); err != nil {
		return nil, err
	}
	normalizeParsedTransaction(&parsed)
	return &parsed, nil
}

func extractJSONObject(value string) (string, error) {
	value = strings.TrimSpace(value)
	start := strings.Index(value, "{")
	end := strings.LastIndex(value, "}")
	if start < 0 || end < start {
		return "", fmt.Errorf("provider output did not contain a JSON object")
	}
	return value[start : end+1], nil
}

func normalizeParsedTransaction(parsed *model.ParsedTransaction) {
	parsed.Intent = strings.ToUpper(strings.TrimSpace(parsed.Intent))
	switch parsed.Intent {
	case "TRANSACTION", "GROUP_TX", "QUERY_BALANCE", "QUERY_REPORT", "HELP", "CHITCHAT", "UNKNOWN":
	default:
		parsed.Intent = "UNKNOWN"
	}

	parsed.Deskripsi = strings.TrimSpace(parsed.Deskripsi)
	parsed.Kategori = strings.TrimSpace(parsed.Kategori)
	parsed.Tipe = strings.ToUpper(strings.TrimSpace(parsed.Tipe))
	parsed.Error = strings.TrimSpace(parsed.Error)
	if parsed.GroupName != nil {
		groupName := strings.TrimSpace(*parsed.GroupName)
		if groupName == "" {
			parsed.GroupName = nil
		} else {
			parsed.GroupName = &groupName
		}
	}

	if parsed.Confidence < 0 {
		parsed.Confidence = 0
	}
	if parsed.Confidence > 1 {
		parsed.Confidence = 1
	}

	if parsed.Intent == "TRANSACTION" || parsed.Intent == "GROUP_TX" {
		if parsed.Deskripsi == "" {
			parsed.Deskripsi = "Transaksi"
		}
		if parsed.Kategori == "" {
			parsed.Kategori = classifyCategory(strings.ToLower(parsed.Deskripsi))
		}
		if parsed.Tipe != "IN" && parsed.Tipe != "OUT" {
			parsed.Tipe = "OUT"
		}
		if parsed.Confidence == 0 {
			parsed.Confidence = 0.8
		}
	}
}

func parseMessageLocally(message string) *model.ParsedTransaction {
	clean := strings.ToLower(strings.TrimSpace(message))
	if clean == "" {
		return &model.ParsedTransaction{Intent: "UNKNOWN", Confidence: 0.2}
	}
	if containsAny(clean, "help", "bantuan", "cara pakai", "format") {
		return &model.ParsedTransaction{Intent: "HELP", Confidence: 1}
	}
	if containsAny(clean, "saldo", "balance") {
		return &model.ParsedTransaction{Intent: "QUERY_BALANCE", Confidence: 1}
	}
	if containsAny(clean, "laporan", "report", "rekap", "pengeluaran bulan", "pengeluaran minggu") {
		return &model.ParsedTransaction{Intent: "QUERY_REPORT", Confidence: 0.95}
	}
	if containsAny(clean, "hai", "halo", "hello") && len(clean) <= 12 {
		return &model.ParsedTransaction{Intent: "CHITCHAT", Confidence: 0.9}
	}

	amount, amountText := parseAmount(clean)
	if amount <= 0 {
		return &model.ParsedTransaction{Intent: "UNKNOWN", Confidence: 0.35}
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
	}
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
