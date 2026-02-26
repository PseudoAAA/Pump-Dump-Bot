package ForBot

type User struct {
	ID int64 `json:"id"`
}

type UserSettings struct {
}

type DepositRecord struct {
	Amount string `json:"amount"`
	Coin   string `json:"coin"`
	Status int    `json:"status"` // 5 — успешно, 0 — в процессе
	TxID   string `json:"txId"`
}
