package types

type MsgInsertBulkWorkerPayload struct {
	Type  string `json:"@type"`
	Nonce struct {
		BlockHeight string `json:"block_height"`
	} `json:"nonce"`
	Sender  string `json:"sender"`
	TopicID string `json:"topic_id"`

	WorkerDataBundles []WorkerDataBundle `json:"worker_data_bundles"`
}

type MsgInsertBulkReputerPayload struct {
	Type                string `json:"@type"`
	Sender              string `json:"sender"`
	TopicID             string `json:"topic_id"`
	ReputerRequestNonce struct {
		WorkerNonce struct {
			BlockHeight string `json:"block_height"`
		} `json:"worker_nonce"`
		ReputerNonce struct {
			BlockHeight string `json:"block_height"`
		} `json:"reputer_nonce"`
	} `json:"reputer_request_nonce"`
	ReputerValueBundles []struct {
		Pubkey      string         `json:"pubkey"`
		Signature   string         `json:"signature"`
		ValueBundle MsgValueBundle `json:"value_bundle"`
	} `json:"reputer_value_bundles"`
}
