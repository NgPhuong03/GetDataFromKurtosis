package model

type Epoch struct {
	Epoch int `json:"epoch" bson:"epoch"`
	Finalized bool `json:"finalized" bson:"finalized"`
	VotingFinalized bool `json:"voting_finalized" bson:"voting_finalized"`
	VotingJustified bool `json:"voting_justified" bson:"voting_justified"`
	Validators int `json:"validators" bson:"validators"`
	ValidatorBalance string `json:"validator_balance" bson:"validator_balance"`
	EligibleEther string `json:"eligible_ether" bson:"eligible_ether"`
	TargetVoted int `json:"target_voted" bson:"target_voted"`
	HeadVoted int `json:"head_voted" bson:"head_voted"`
	TotalVoted int `json:"total_voted" bson:"total_voted"`
	VoteParticipation int `json:"vote_participation" bson:"vote_participation"`
	Attestations int `json:"attestations" bson:"attestations"`
	Deposits int `json:"deposits" bson:"deposits"`
	DepositsAmount string `json:"deposits_amount" bson:"deposits_amount"`
	ProposerSlashings int `json:"proposer_slashings" bson:"proposer_slashings"`
	AttesterSlashings int `json:"attester_slashings" bson:"attester_slashings"`
	Exits int `json:"exits" bson:"exits"`
	WithdrawalsCount int `json:"withdrawals_count" bson:"withdrawals_count"`
	WithdrawalsAmount string `json:"withdrawals_amount" bson:"withdrawals_amount"`
	BlsChanges int `json:"bls_changes" bson:"bls_changes"`
	SyncParticipation int `json:"sync_participation" bson:"sync_participation"`
	ProposedBlocks int `json:"proposed_blocks" bson:"proposed_blocks"`
	MissedBlocks int `json:"missed_blocks" bson:"missed_blocks"`
	OrphanedBlocks int `json:"orphaned_blocks" bson:"orphaned_blocks"`
}