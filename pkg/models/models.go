package models

type Order struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float32 `json:"accrual"`
	UploadedAt string  `json:"uploaded_at"`
}

type Balance struct {
	Current  float32 `json:"current"`
	Withdraw float32 `json:"withdrawn"`
}

type WithdrawInfo struct {
	Order       string  `json:"order"`
	Sum         float32 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

type Withdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

type AuthModel struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AccrualModel struct {
	OrderNumber string  `json:"order"`
	Status      string  `json:"status"`
	Accrual     float32 `json:"accrual"`
}
