package payments

type TopupResponse struct {
	PaymentID string `json:"payment_id"`
	Amount    int    `json:"amount"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}
