package model

type Validator struct {
  Index int `json:"index" bson:"index"`
  Name string `json:"name" bson:"name"`
  PublicKey string `json:"public_key" bson:"public_key"`
  Balance string `json:"balance" bson:"balance"`
  EffectiveBalance string `json:"effective_balance" bson:"effective_balance"`
  Status string `json:"status" bson:"status"`
  ActivationTime string `json:"activation_time" bson:"activation_time"`
  WithdrawalAddress string `json:"withdrawal_address" bson:"withdrawal_address"`
  WithdrawalCredentials string `json:"withdrawal_credentials" bson:"withdrawal_credentials"`
  ValidatorLiveness int `json:"validator_liveness" bson:"validator_liveness"`
  ValidatorLivenessMax int `json:"validator_liveness_max" bson:"validator_liveness_max"`
  Epoch int `json:"epoch" bson:"epoch"`
}