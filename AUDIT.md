# Codebase Audit — money-tracker-service

Audit dilakukan pada seluruh codebase. Temuan dikelompokkan berdasarkan tingkat keparahan.

---

## CRITICAL

### 1. Invalid OpenAI model name
**File:** `internal/config/config.go:70`  
Model default di-set ke `"gpt-5-4"` — model ini tidak ada. Jika OpenAI API key dikonfigurasi, semua request ke OpenAI akan gagal.  
**Fix:** Ganti ke nama model yang valid, misalnya `"gpt-4o"`.

---

## MEDIUM

### 2. `SendExpiryReminders()` tidak diimplementasi
**File:** `internal/modules/balance/service.go:34-36`  
Fungsi ini dipanggil dari cron di `cmd/server/main.go:159` tapi body-nya hanya `return nil` — tidak ada logika pengiriman reminder apapun.

### 3. Nilai hardcoded di admin service
**File:** `internal/modules/admin/service.go:34-43` dan `128-131`  
Biaya infrastruktur dan split profit semuanya hardcoded langsung di kode:
- `infraCost := 478600`
- `referralCost := 125000`
- Profit split: `"daniel_75"` → 75%, `"teman_25"` → 25%
- VPS, Fonnte, Domain, OpenAI cost (line 128-131) — nilai berbeda dari yang di atas, inkonsisten

Nilai-nilai ini seharusnya ada di config atau database.

### 4. Stats admin selalu 0 / nil
**File:** `internal/modules/admin/service.go:71-77`  
Field berikut selalu return nilai statis, tidak dihitung dari database:
```go
"total_wa_messages_this_month": 0,
"total_ai_calls_this_month":    0,
"ai_cost_this_month":           0,
"registered_via_referral":      nil,
```

### 5. Validasi file upload proof tidak ada
**File:** `internal/modules/payments/handler.go:63-97`  
Fungsi `saveProof()` mengembalikan `nil, nil` ketika file tidak ada — tidak ada error yang di-return ke client. User bisa submit topup tanpa bukti transfer dan tidak ada feedback error.

### 6. Referral commission hardcoded
**File:** `internal/modules/referral/service.go:32` dan `79`  
Nilai komisi `5000` di-hardcode di dua tempat berbeda. Seharusnya configurable.

---

## LOW

### 7. Field `account_id` tidak di-fetch di beberapa query
Field ini ada di model tapi tidak di-SELECT di query berikut:
- `internal/modules/dashboard/repository.go:102-110` — `TransactionsForPeriod()`
- `internal/modules/transactions/repository.go:85,98-99`
- `internal/modules/admin/repository.go:85`
- `internal/modules/groups/repository.go:124-126` — `ListTransactionsForReport()`

Ini bisa menyebabkan field `AccountID` selalu `nil` pada response meski data ada di DB.

### 8. `IsGracePeriod` selalu `false`
**File:** `internal/modules/balance/handler.go:41`  
Field ini selalu di-set `false` secara hardcoded, tidak pernah dikalkulasi. Fitur grace period tidak berfungsi.

### 9. Webhook handler ignore binding error
**File:** `internal/modules/webhook/handler.go:20`  
```go
_ = c.Bind(&req)
```
Error parsing request webhook diabaikan sepenuhnya — jika format request salah, eksekusi tetap lanjut dengan struct kosong.

### 10. Webhook repository suppress semua DB errors
**File:** `internal/modules/webhook/repository.go:33-38`  
```go
_, _ = r.db.ExecContext(ctx, ...)
```
Semua error saat logging pesan WA ke database diabaikan.

### 11. Cron errors diabaikan di main
**File:** `cmd/server/main.go:156-159`  
Semua return value dari fungsi cron di-discard dengan `_`. Kegagalan cron tidak ter-log sama sekali.

### 12. Async webhook handler pakai `context.Background()`
**File:** `internal/modules/webhook/handler.go:25`  
```go
go h.service.Handle(context.Background(), req, token)
```
Request context di-drop saat goroutine di-spawn — trace dan logging context hilang.

---

## Summary

| Severity | Jumlah |
|----------|--------|
| Critical | 1 |
| Medium   | 5 |
| Low      | 6 |
| **Total**| **12** |

**Prioritas utama:**
1. Fix nama model OpenAI (akan langsung break fitur AI)
2. Implementasi `SendExpiryReminders()`
3. Tambah validasi file upload di payments
4. Pindahkan hardcoded costs ke config
