package model

type Slot struct {
	Slot int `json:"slot" bson:"slot"`
	Epoch int `json:"epoch" bson:"epoch"`
	Time string `json:"time" bson:"time"`
	Finalized bool `json:"finalized" bson:"finalized"`
	Scheduled bool `json:"scheduled" bson:"scheduled"`
	Status string `json:"status" bson:"status"`
	Proposer int `json:"proposer" bson:"proposer"`
	ProposerName string `json:"proposer_name" bson:"proposer_name"`
	AttestationCount int `json:"attestation_count" bson:"attestation_count"`
	DepositCount int `json:"deposit_count" bson:"deposit_count"`
	ExitCount int `json:"exit_count" bson:"exit_count"`
	ProposerSlashingCount int `json:"proposer_slashing_count" bson:"proposer_slashing_count"`
	AttesterSlashingCount int `json:"attester_slashing_count" bson:"attester_slashing_count"`
	SyncAggregateParticipation int `json:"sync_aggregate_participation" bson:"sync_aggregate_participation"`
	EthTransactionCount int `json:"eth_transaction_count" bson:"eth_transaction_count"`
	BlobCount int `json:"blob_count" bson:"blob_count"`
	WithEthBlock bool `json:"with_eth_block" bson:"with_eth_block"`
	EthBlockNumber int `json:"eth_block_number" bson:"eth_block_number"`
	Graffiti string `json:"graffiti" bson:"graffiti"`
	GraffitiText string `json:"graffiti_text" bson:"graffiti_text"`
	ElExtraData string `json:"el_extra_data" bson:"el_extra_data"`
	GasUsed int `json:"gas_used" bson:"gas_used"`
	GasLimit int `json:"gas_limit" bson:"gas_limit"`
	BlockSize int `json:"block_size" bson:"block_size"`
	BlockRoot string `json:"block_root" bson:"block_root"`
	ParentRoot string `json:"parent_root" bson:"parent_root"`
	StateRoot string `json:"state_root" bson:"state_root"`
	RecvDelay int `json:"recv_delay" bson:"recv_delay"`
	IsMevBlock bool `json:"is_mev_block" bson:"is_mev_block"`
}