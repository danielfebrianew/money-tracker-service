package service

import (
	"context"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type GroupService struct {
	store        *repository.Store
	cache        *cache.Cache
	transactions *TransactionService
}

func NewGroupService(store *repository.Store, cache *cache.Cache, transactions *TransactionService) *GroupService {
	return &GroupService{store: store, cache: cache, transactions: transactions}
}

func (s *GroupService) Create(ctx context.Context, ownerID, name string) (*model.BudgetGroup, []model.GroupMemberView, error) {
	count, err := s.store.CountOwnedGroups(ctx, ownerID)
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
	if err := s.store.CreateGroupWithOwner(ctx, group); err != nil {
		return nil, nil, err
	}
	members, err := s.store.ListGroupMembers(ctx, group.ID)
	return &group, members, err
}

func (s *GroupService) List(ctx context.Context, userID string) ([]model.GroupListItem, error) {
	return s.store.ListGroups(ctx, userID)
}

func (s *GroupService) Invite(ctx context.Context, requesterID, groupID, phone string) (*model.GroupMemberView, error) {
	member, err := s.store.GetMembership(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}
	if member.Role != "owner" {
		return nil, apperror.New(apperror.ErrForbidden, "Hanya owner yang bisa mengundang member")
	}
	user, err := s.store.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "User dengan nomor tersebut tidak ditemukan")
	}
	count, err := s.store.CountGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if count >= 5 {
		return nil, apperror.New(apperror.ErrValidation, "Maksimal 5 member per grup")
	}
	if err := s.store.AddGroupMember(ctx, groupID, user.ID); err != nil {
		return nil, err
	}
	return &model.GroupMemberView{UserID: user.ID, Name: user.Name, Phone: user.Phone, Role: "member"}, nil
}

func (s *GroupService) CreateTransaction(ctx context.Context, userID, groupID, deskripsi string, jumlah int, tipe string) (*model.Transaction, error) {
	return s.transactions.CreateGroupTransaction(ctx, userID, groupID, model.CreateTransactionInput{
		Deskripsi: deskripsi,
		Jumlah:    jumlah,
		Tipe:      tipe,
		Source:    "dashboard",
	})
}

func (s *GroupService) Report(ctx context.Context, userID, groupID, month string) (*model.GroupReport, error) {
	if _, err := s.store.GetMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}
	group, err := s.store.GetGroup(ctx, groupID)
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
	start, end, err := repository.MonthRange(month)
	if err != nil {
		return nil, err
	}
	params := model.TransactionListParams{
		GroupID: &groupID,
		From:    &start,
		To:      &end,
		Page:    1,
		PerPage: 10000,
	}
	items, _, err := s.store.ListTransactions(ctx, params)
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
	members, _ := s.store.ListGroupMembers(ctx, groupID)
	for _, member := range members {
		total := memberTotals[member.Phone]
		percent := 0.0
		if report.TotalOut > 0 {
			percent = repository.Round1(float64(total) / float64(report.TotalOut) * 100)
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
