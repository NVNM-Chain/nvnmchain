package chainsuite

import (
	"time"
)

type Metadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Metadata    string `json:"metadata"`
}

type ConsumerChain struct {
	ChainID            string   `json:"chain_id"`
	ClientID           string   `json:"client_id"`
	TopN               int      `json:"top_N"`
	MinPowerInTopN     string   `json:"min_power_in_top_N"`
	ValidatorsPowerCap int      `json:"validators_power_cap"`
	ValidatorSetCap    int      `json:"validator_set_cap"`
	Allowlist          []string `json:"allowlist"`
	Denylist           []string `json:"denylist"`
	Phase              string   `json:"phase"`
	Metadata           Metadata `json:"metadata"`
	MinStake           string   `json:"min_stake"`
	AllowInactiveVals  bool     `json:"allow_inactive_vals"`
	ConsumerID         string   `json:"consumer_id"`
}

type Pagination struct {
	NextKey interface{} `json:"next_key"`
	Total   string      `json:"total"`
}

type ListConsumerChainsResponse struct {
	Chains     []ConsumerChain `json:"chains"`
	Pagination Pagination      `json:"pagination"`
}

type ConsumerResponse struct {
	ChainID            string             `json:"chain_id"`
	ConsumerID         string             `json:"consumer_id"`
	InitParams         InitParams         `json:"init_params"`
	Metadata           Metadata           `json:"metadata"`
	OwnerAddress       string             `json:"owner_address"`
	Phase              string             `json:"phase"`
	PowerShapingParams PowerShapingParams `json:"power_shaping_params"`
	InfractionParams   InfractionParams   `json:"infraction_parameters"`
}

type InitParams struct {
	BinaryHash                        string        `json:"binary_hash"`
	BlocksPerDistributionTransmission string        `json:"blocks_per_distribution_transmission"`
	CCVTimeoutPeriod                  string        `json:"ccv_timeout_period"`
	ConsumerRedistributionFraction    string        `json:"consumer_redistribution_fraction"`
	DistributionTransmissionChannel   string        `json:"distribution_transmission_channel"`
	GenesisHash                       string        `json:"genesis_hash"`
	HistoricalEntries                 string        `json:"historical_entries"`
	InitialHeight                     InitialHeight `json:"initial_height"`
	SpawnTime                         time.Time     `json:"spawn_time"`
	TransferTimeoutPeriod             string        `json:"transfer_timeout_period"`
	UnbondingPeriod                   string        `json:"unbonding_period"`
}

type InitialHeight struct {
	RevisionHeight string `json:"revision_height"`
	RevisionNumber string `json:"revision_number"`
}

type PowerShapingParams struct {
	AllowInactiveVals  bool     `json:"allow_inactive_vals"`
	Allowlist          []string `json:"allowlist"`
	Denylist           []string `json:"denylist"`
	MinStake           string   `json:"min_stake"`
	TopN               int      `json:"top_N"`
	ValidatorSetCap    int      `json:"validator_set_cap"`
	ValidatorsPowerCap int      `json:"validators_power_cap"`
}

type InfractionParams struct {
	DoubleSign SlashJailParams `json:"double_sign"`
	Downtime   SlashJailParams `json:"downtime"`
}

type SlashJailParams struct {
	SlashFraction string `json:"slash_fraction"`
	JailDuration  string `json:"jail_duration"`
}

type ProviderInfoResponse struct {
	Consumer ChainDetails `json:"consumer"`
	Provider ChainDetails `json:"provider"`
}

type ChainDetails struct {
	ChainID      string `json:"chainID"`
	ClientID     string `json:"clientID"`
	ConnectionID string `json:"connectionID"`
	ChannelID    string `json:"channelID"`
}
