package groups

import (
	"context"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	transactions "money-management-service/internal/modules/transactions"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
)

type Service struct {
	repository   *Repository
	cache        *cache.Cache
	transactions *transactions.Service
}

func NewService(repository *Repository, cache *cache.Cache, transactions *transactions.Service) *Service {
	return &Service{repository: repository, cache: cache, transactions: transactions}
}

func (s *Service) Create(ctx context.Context, ownerID, name string) (*model.BudgetGroup, []model.GroupMemberView, error) {
	count, err := s.repository.CountOwned(ctx, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if count >= 3 {
		return nil, nil, apperror.New(apperror.ErrValidation, "Maksimal 3 grup per akun")
	}
	group := model.BudgetGroup{
		ID:        ids.New("grp"),
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repository.CreateWithOwner(ctx, group); err != nil {
		return nil, nil, err
	}
	members, err := s.repository.ListMembers(ctx, group.ID)
	return &group, members, err
}

func (s *Service) List(ctx context.Context, userID string) ([]model.GroupListItem, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Invite(ctx context.Context, requesterID, groupID, phone string) (*model.GroupMemberView, error) {
	member, err := s.repository.GetMembership(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}
	if member.Role != "owner" {
		return nil, apperror.New(apperror.ErrForbidden, "Hanya owner yang bisa mengundang member")
	}
	user, err := s.repository.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "User dengan nomor tersebut tidak ditemukan")
	}
	count, err := s.repository.CountMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if count >= 5 {
		return nil, apperror.New(apperror.ErrValidation, "Maksimal 5 member per grup")
	}
	if err := s.repository.AddMember(ctx, groupID, user.ID); err != nil {
		return nil, err
	}
	return &model.GroupMemberView{UserID: user.ID, Name: user.Name, Phone: user.Phone, Role: "member"}, nil
}

func (s *Service) CreateTransaction(ctx context.Context, userID, groupID, deskripsi string, jumlah int, tipe string) (*model.Transaction, error) {
	return s.transactions.CreateGroupTransaction(ctx, userID, groupID, model.CreateTransactionInput{
		Deskripsi: deskripsi,
		Jumlah:    jumlah,
		Tipe:      tipe,
		Source:    "dashboard",
	})
}

func (s *Service) Report(ctx context.Context, userID, groupID, month string) (*model.GroupReport, error) {
	if _, err := s.repository.GetMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}
	group, err := s.repository.Get(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	key := "group:" + groupID + ":report:" + month
	var cached model.GroupReport
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	start, end, err := monthRange(month)
	if err != nil {
		return nil, err
	}
	items, err := s.repository.ListTransactionsForReport(ctx, groupID, start, end)
	if err != nil {
		return nil, err
	}
	categories := map[string]model.CategoryTotal{}
	memberTotals := map[string]int{}
	for _, item := range items {
		if item.Tipe != "OUT" || item.RecordedBy == nil {
			continue
		}
		memberTotals[*item.RecordedBy] += item.Jumlah
		cat := categories[item.Kategori]
		cat.Kategori = item.Kategori
		cat.Total += item.Jumlah
		categories[item.Kategori] = cat
	}
	report := &model.GroupReport{GroupName: group.Name, Month: month}
	for _, total := range memberTotals {
		report.TotalOut += total
	}
	members, _ := s.repository.ListMembers(ctx, groupID)
	for _, member := range members {
		total := memberTotals[member.Phone]
		percent := 0.0
		if report.TotalOut > 0 {
			percent = round1(float64(total) / float64(report.TotalOut) * 100)
		}
		report.ByMember = append(report.ByMember, model.GroupMemberTotal{
			Phone:   member.Phone,
			Name:    member.Name,
			Total:   total,
			Percent: percent,
		})
	}
	for _, cat := range categories {
		report.ByKategori = append(report.ByKategori, cat)
	}
	s.cache.SetJSON(ctx, key, report, 5*time.Minute)
	return report, nil
}
