package goals

type CreateInput struct {
	Name         string `json:"name"`
	TargetAmount int    `json:"target_amount"`
	Deadline     string `json:"deadline"`
	Icon         string `json:"icon"`
	Color        string `json:"color"`
}

type UpdateInput struct {
	Name         *string `json:"name"`
	TargetAmount *int    `json:"target_amount"`
	Deadline     *string `json:"deadline"`
	Icon         *string `json:"icon"`
	Color        *string `json:"color"`
}

type ContributeInput struct {
	Amount int `json:"amount"`
}
