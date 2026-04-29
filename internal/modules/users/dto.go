package users

type UpdateRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Timezone *string `json:"timezone"`
}
