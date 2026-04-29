package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"money-management-service/internal/config"
)

type FonnteService interface {
	SendReply(ctx context.Context, req FonnteSendRequest) (*FonnteSendResponse, error)
	SendTyping(ctx context.Context, target string, durationSec int) error
	ValidateWebhookToken(token string) bool
}

type FonnteSendRequest struct {
	Target   string `json:"target"`
	Message  string `json:"message"`
	Typing   bool   `json:"typing,omitempty"`
	Duration int    `json:"duration,omitempty"`
	InboxID  string `json:"inboxid,omitempty"`
}

type FonnteSendResponse struct {
	Status  bool     `json:"status"`
	Detail  string   `json:"detail"`
	ID      []string `json:"id"`
	Process string   `json:"process"`
}

type FonnteClient struct {
	cfg        config.Config
	httpClient *http.Client
}

func NewFonnteClient(cfg config.Config) *FonnteClient {
	return &FonnteClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (f *FonnteClient) ValidateWebhookToken(token string) bool {
	return f.cfg.FonnteWebhookToken == "" || token == f.cfg.FonnteWebhookToken
}

func (f *FonnteClient) SendReply(ctx context.Context, req FonnteSendRequest) (*FonnteSendResponse, error) {
	if f.cfg.FonnteToken == "" {
		return &FonnteSendResponse{Status: true, Detail: "fonnte disabled"}, nil
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.fonnte.com/send", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", f.cfg.FonnteToken)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result FonnteSendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (f *FonnteClient) SendTyping(ctx context.Context, target string, durationSec int) error {
	if f.cfg.FonnteToken == "" {
		return nil
	}
	body, err := json.Marshal(map[string]interface{}{
		"target":   target,
		"duration": durationSec,
	})
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.fonnte.com/typing", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", f.cfg.FonnteToken)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
