package service

import (
	"context"
	"fmt"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	"money-management-service/internal/model"
	"money-management-service/internal/repository"
)

type WebhookService struct {
	cfg          config.Config
	store        *repository.Store
	cache        *cache.Cache
	parser       OpenAIService
	fonnte       FonnteService
	transactions *TransactionService
}

type FonnteWebhookPayload struct {
	Device    string `json:"device"`
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	Text      string `json:"text"`
	Member    string `json:"member"`
	Name      string `json:"name"`
	Location  string `json:"location"`
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	Extension string `json:"extension"`
	PollName  string `json:"pollname"`
	Choices   string `json:"choices"`
	InboxID   string `json:"inboxid"`
	Timestamp string `json:"timestamp"`
	Token     string `json:"token"`
}

func NewWebhookService(cfg config.Config, store *repository.Store, cache *cache.Cache, parser OpenAIService, fonnte FonnteService, transactions *TransactionService) *WebhookService {
	return &WebhookService{cfg: cfg, store: store, cache: cache, parser: parser, fonnte: fonnte, transactions: transactions}
}

func (s *WebhookService) Handle(ctx context.Context, payload FonnteWebhookPayload, token string) {
	if token == "" {
		token = payload.Token
	}
	if !s.fonnte.ValidateWebhookToken(token) {
		return
	}
	message := payload.Message
	if message == "" {
		message = payload.Text
	}
	user, err := s.store.GetUserByPhone(ctx, payload.Sender)
	if err != nil {
		s.reply(ctx, payload, "Nomor kamu belum terdaftar. Daftar di "+s.cfg.AppURL)
		return
	}
	s.store.LogWAMessage(ctx, &user.ID, payload.Sender, message)
	if !user.IsActive {
		s.reply(ctx, payload, "Akun kamu dinonaktifkan. Hubungi admin.")
		return
	}
	balance, err := s.store.GetBalance(ctx, user.ID)
	if err != nil || balance.Balance <= 0 {
		s.reply(ctx, payload, "Saldo habis. Top-up di dashboard: "+s.cfg.AppURL)
		return
	}
	if !s.checkMessageQuota(ctx, user.ID) {
		s.reply(ctx, payload, "Kamu sudah mencapai batas pesan hari ini.")
		return
	}
	_ = s.fonnte.SendTyping(ctx, payload.Sender, 2)
	parsed, err := s.parser.ParseMessage(ctx, message)
	if err != nil || parsed == nil {
		s.reply(ctx, payload, "Maaf, aku gak ngerti. Contoh: makan siang 25rb")
		return
	}
	switch parsed.Intent {
	case "TRANSACTION":
		s.handleTransaction(ctx, user, payload, parsed)
	case "GROUP_TX":
		s.handleGroupTransaction(ctx, user, payload, parsed)
	case "QUERY_BALANCE":
		s.replyBalance(ctx, user, payload, balance)
	case "QUERY_REPORT":
		s.replyReport(ctx, user, payload)
	case "HELP":
		s.reply(ctx, payload, "Cara Pakai:\n\nCatat: makan siang 25rb\nPemasukan: gaji 5jt\nLaporan: rekap bulan ini\nSaldo: saldo\nGrup: grup keluarga makan 150rb")
	case "CHITCHAT":
		s.reply(ctx, payload, "Hai! Aku bisa bantu catat keuangan. Contoh: makan siang 25rb")
	default:
		msg := "Maaf, aku gak ngerti. Contoh: makan siang 25rb"
		if parsed.Error != "" {
			msg = parsed.Error
		}
		s.reply(ctx, payload, msg)
	}
}

func (s *WebhookService) handleTransaction(ctx context.Context, user *model.User, payload FonnteWebhookPayload, parsed *model.ParsedTransaction) {
	if parsed.Confidence < 0.8 {
		s.reply(ctx, payload, fmt.Sprintf("Maksudnya: %s Rp%d (%s)?\n\nBalas ya untuk konfirmasi.", parsed.Deskripsi, parsed.Jumlah, parsed.Kategori))
		return
	}
	tx, err := s.transactions.CreateFromWA(ctx, user, parsed)
	if err != nil {
		s.reply(ctx, payload, "Transaksi belum bisa dicatat. Coba lagi sebentar.")
		return
	}
	summary, _ := s.store.GetTransactionSummary(ctx, user.ID, time.Now().Format("2006-01"))
	totalOut := 0
	if summary != nil {
		totalOut = summary.TotalOut
	}
	s.reply(ctx, payload, fmt.Sprintf("Sudah Dicatat!\n\n%s\nRp%d\n%s · %s\n\nTotal pengeluaran bulan ini: Rp%d", tx.Tipe, tx.Jumlah, tx.Kategori, tx.Deskripsi, totalOut))
}

func (s *WebhookService) handleGroupTransaction(ctx context.Context, user *model.User, payload FonnteWebhookPayload, parsed *model.ParsedTransaction) {
	if parsed.GroupName == nil {
		s.reply(ctx, payload, "Nama grup belum jelas. Contoh: grup keluarga makan 150rb")
		return
	}
	group, err := s.store.FindGroupByNameForUser(ctx, user.ID, *parsed.GroupName)
	if err != nil {
		s.reply(ctx, payload, "Grup tidak ditemukan.")
		return
	}
	tx, err := s.transactions.CreateGroupTransactionFromWA(ctx, user, group.ID, parsed)
	if err != nil {
		s.reply(ctx, payload, "Transaksi grup belum bisa dicatat. Coba lagi sebentar.")
		return
	}
	s.reply(ctx, payload, fmt.Sprintf("Tercatat ke grup %s: %s Rp%d", group.Name, tx.Kategori, tx.Jumlah))
}

func (s *WebhookService) replyBalance(ctx context.Context, user *model.User, payload FonnteWebhookPayload, balance *model.UserBalance) {
	summary, _ := s.store.GetTransactionSummary(ctx, user.ID, time.Now().Format("2006-01"))
	monthOut := 0
	if summary != nil {
		monthOut = summary.TotalOut
	}
	s.reply(ctx, payload, fmt.Sprintf("Saldo: Rp%d\nBulan ini: -Rp%d", balance.Balance, monthOut))
}

func (s *WebhookService) replyReport(ctx context.Context, user *model.User, payload FonnteWebhookPayload) {
	summary, _ := s.store.GetTransactionSummary(ctx, user.ID, time.Now().Format("2006-01"))
	if summary == nil {
		s.reply(ctx, payload, "Belum ada laporan bulan ini.")
		return
	}
	s.reply(ctx, payload, fmt.Sprintf("Laporan bulan ini:\nMasuk: Rp%d\nKeluar: Rp%d\nSaldo: Rp%d\nTotal transaksi: %d", summary.TotalIn, summary.TotalOut, summary.Saldo, summary.TotalTransactions))
}

func (s *WebhookService) reply(ctx context.Context, payload FonnteWebhookPayload, message string) {
	_, _ = s.fonnte.SendReply(ctx, FonnteSendRequest{
		Target:  payload.Sender,
		Message: message,
		InboxID: payload.InboxID,
	})
}

func (s *WebhookService) checkMessageQuota(ctx context.Context, userID string) bool {
	client := s.cache.Client()
	if client == nil {
		return true
	}
	hourKey := "wa:quota:" + userID + ":hour:" + time.Now().Format("2006010215")
	dayKey := "wa:quota:" + userID + ":day:" + time.Now().Format("20060102")
	hour, _ := client.Incr(ctx, hourKey).Result()
	day, _ := client.Incr(ctx, dayKey).Result()
	if hour == 1 {
		client.Expire(ctx, hourKey, time.Hour)
	}
	if day == 1 {
		client.Expire(ctx, dayKey, 24*time.Hour)
	}
	return hour <= 30 && day <= 100
}
