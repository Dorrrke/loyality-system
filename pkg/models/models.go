package models

type Order struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadedAt string `json:"uploaded_at"`
}

type Balance struct {
	Current  float32 `json:"current"`
	Withdraw int     `json:"withdrawn"`
}

type Withdraw struct {
	Order       string `json:"order"`
	Sum         int    `json:"sum"`
	ProcessedAt string `json:"processed_at"`
}

type AuthModel struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
