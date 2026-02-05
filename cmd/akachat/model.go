package main

type User struct {
	ID         string `json:"id"`
	Token      string `json:"token"`
	PrivateKey string `json:"privateKey"`
}

type SignedTxRequest struct {
	CacheKey string `json:"cacheKey"`
	SignedTx string `json:"signedTx"`
	UserID   string `json:"userId"`
}

type APIResponse[T any] struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	ErrDlt  string `json:"errDlt"`
	Data    T      `json:"data"`
}

type UnsignedTx struct {
	To       string `json:"to"`
	From     string `json:"from"`
	Data     string `json:"data"`
	Value    string `json:"value"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Nonce    string `json:"nonce"`
	ChainID  string `json:"chainId"`
	CacheKey string `json:"cacheKey"`
}

type Fee struct {
	Currency  string `json:"currency"`
	Estimated string `json:"estimated"`
	Formatted string `json:"formatted"`
}

type Meta struct {
	Action    string `json:"action"`
	Chain     string `json:"chain"`
	ExpiresAt int64  `json:"expiresAt"`
}

type UnsignedTxData struct {
	Network    string     `json:"network"`
	UnsignedTx UnsignedTx `json:"unsignedTx"`
	Fee        Fee        `json:"fee"`
	Meta       Meta       `json:"meta"`
}

type SignedTxResult struct {
	TxHash            string      `json:"txHash"`
	BlockNumber       int64       `json:"blockNumber"`
	BlockHash         string      `json:"blockHash"`
	Status            int         `json:"status"`
	GasUsed           int64       `json:"gasUsed"`
	CumulativeGasUsed int64       `json:"cumulativeGasUsed"`
	ContractAddress   string      `json:"contractAddress"`
	Logs              interface{} `json:"logs"`
	EnvelopeID        int64       `json:"envelopeId"`
}

type PayloadCreate struct {
	EnvelopeType        string `json:"envelopeType"`
	Token               string `json:"token"`
	TotalClaims         int    `json:"totalClaims"`
	AmountPerClaimOrPot int    `json:"AmountPerClaimOrPot"`
	Value               int    `json:"value"`
	Chain               string `json:"chain"`
	GroupID             string `json:"groupID"`
	Remarks             string `json:"remarks"`
	ThemeID             int    `json:"themeID"`
	ToUserID            string `json:"toUserID"`
	UserID              string `json:"userID"`
}

type PayloadClaim struct {
	Chain          string `json:"chain"`
	UserID         string `json:"userID"`
	GroupID        string `json:"groupID"`
	EnvelopeID     int    `json:"envelopeID"`
	ConversationID string `json:"conversationID"`
	Seq            int    `json:"seq"`
	Status         string `json:"status"`
}

type PayloadSignedTx struct {
	RawTransaction string `json:"rawTransaction"`
	TxHash         string `json:"txHash"`
	Chain          string `json:"chain"`
	CacheKey       string `json:"cacheKey"`
	Action         string `json:"action"`
}
