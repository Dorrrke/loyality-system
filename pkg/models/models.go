package models

type Order struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadedAt string `json:"uploaded_at"`
}

type Balance struct {
	Current  int `json:"current"`
	Withdraw int `json:"withdrawn"`
}

type WithdrawInfo struct {
	Order       string `json:"order"`
	Sum         int    `json:"sum"`
	ProcessedAt string `json:"processed_at"`
}

type Withdraw struct {
	Order string `json:"order"`
	Sum   int    `json:"sum"`
}

type AuthModel struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AccrualModel struct {
	OrderNumber string `json:"order"`
	Status      string `json:"status"`
	Accrual     int    `json:"accrual"`
}
