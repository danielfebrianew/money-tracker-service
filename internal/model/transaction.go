package model

import "time"

type Transaction struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	GroupID    *string   `json:"group_id,omitempty" db:"group_id"`
	WalletID  *string   `json:"wallet_id,omitempty" db:"wallet_id"`
	Jumlah     int       `json:"jumlah" db:"jumlah"`
	Deskripsi  string    `json:"deskripsi" db:"deskripsi"`
	Kategori   string    `json:"kategori" db:"kategori"`
	Tipe       string    `json:"tipe" db:"tipe"`
	Source     string    `json:"source" db:"source"`
	RecordedBy *string   `json:"recorded_by,omitempty" db:"recorded_by"`
	Confidence *float64  `json:"confidence,omitempty" db:"confidence"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type TransactionResponse struct {
	Transaction
	WalletName *string `json:"wallet_name,omitempty" db:"wallet_name"`
}

type ParsedTransaction struct {
	Intent     string  `json:"intent"`
	Jumlah     int     `json:"jumlah,omitempty"`
	Deskripsi  string  `json:"deskripsi,omitempty"`
	Kategori   string  `json:"kategori,omitempty"`
	Tipe       string  `json:"tipe,omitempty"`
	GroupName  *string `json:"group_name,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Error      string  `json:"error,omitempty"`
}

type TransactionFilters struct {
	Page      int
	PerPage   int
	Tipe      string
	Kategori  string
	WalletID *string
	From      *time.Time
	To        *time.Time
	Search    string
}

type TransactionListParams struct {
	UserID    string
	GroupID   *string
	WalletID *string
	Tipe      *string
	Kategori  *string
	From      *time.Time
	To        *time.Time
	Search    *string
	Page      int
	PerPage   int
}

type CreateTransactionInput struct {
	Jumlah    int     `json:"jumlah"`
	Deskripsi string  `json:"deskripsi"`
	Kategori  string  `json:"kategori"`
	Tipe      string  `json:"tipe"`
	Source    string  `json:"source"`
	WalletID *string `json:"wallet_id"`
}

type CategoryTotal struct {
	Kategori string  `json:"kategori" db:"kategori"`
	Total    int     `json:"total" db:"total"`
	Count    int     `json:"count,omitempty" db:"count"`
	Percent  float64 `json:"percent,omitempty" db:"percent"`
}

type DailyTrend struct {
	Date string `json:"date" db:"date"`
	In   int    `json:"in" db:"total_in"`
	Out  int    `json:"out" db:"total_out"`
}

type DashboardSummary struct {
	Month             string `json:"month"`
	TotalIn           int    `json:"total_in"`
	TotalOut          int    `json:"total_out"`
	Saldo             int    `json:"saldo"`
	TotalTransactions int    `json:"total_transactions"`
	Comparison        struct {
		PrevMonthOut  int     `json:"prev_month_out"`
		ChangePercent float64 `json:"change_percent"`
	} `json:"comparison"`
}

type ChartData struct {
	Month      string          `json:"month"`
	ByKategori []CategoryTotal `json:"by_kategori"`
	DailyTrend []DailyTrend    `json:"daily_trend"`
}

type PeriodReport struct {
	Period       string          `json:"period"`
	StartDate    string          `json:"start_date"`
	EndDate      string          `json:"end_date"`
	TotalIn      int             `json:"total_in"`
	TotalOut     int             `json:"total_out"`
	Saldo        int             `json:"saldo"`
	ByKategori   []CategoryTotal `json:"by_kategori"`
	Transactions []Transaction   `json:"transactions"`
}
