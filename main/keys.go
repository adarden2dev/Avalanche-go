// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

const (
	defaultString                   = "default"
	configFileKey                   = "config-file"
	versionKey                      = "version"
	genesisConfigFileKey            = "genesis"
	networkNameKey                  = "network-id"
	txFeeKey                        = "tx-fee"
	creationTxFeeKey                = "creation-tx-fee"
	uptimeRequirementKey            = "uptime-requirement"
	minValidatorStakeKey            = "min-validator-stake"
	maxValidatorStakeKey            = "max-validator-stake"
	minDelegatorStakeKey            = "min-delegator-stake"
	minDelegatorFeeKey              = "min-delegation-fee"
	minStakeDurationKey             = "min-stake-duration"
	maxStakeDurationKey             = "max-stake-duration"
	stakeMintingPeriodKey           = "stake-minting-period"
	assertionsEnabledKey            = "assertions-enabled"
	signatureVerificationEnabledKey = "signature-verification-enabled"
	dbEnabledKey                    = "db-enabled"
	dbDirKey                        = "db-dir"
	publicIPKey                     = "public-ip"
	dynamicUpdateDurationKey        = "dynamic-update-duration"
	dynamicPublicIPResolverKey      = "dynamic-public-ip"
	connMeterResetDurationKey       = "conn-meter-reset-duration"
	connMeterMaxConnsKey            = "conn-meter-max-conns"
	httpHostKey                     = "http-host"
	httpPortKey                     = "http-port"
	httpsEnabledKey                 = "http-tls-enabled"
	httpsKeyFileKey                 = "http-tls-key-file"
	httpsCertFileKey                = "http-tls-cert-file"
	apiAuthRequiredKey              = "api-auth-required"
	apiAuthPasswordKey              = "api-auth-password" // #nosec G101
	bootstrapIPsKey                 = "bootstrap-ips"
	bootstrapIDsKey                 = "bootstrap-ids"
	stakingPortKey                  = "staking-port"
	stakingEnabledKey               = "staking-enabled"
	p2pTLSEnabledKey                = "p2p-tls-enabled"
	stakingKeyPathKey               = "staking-tls-key-file"
	stakingCertPathKey              = "staking-tls-cert-file"
	stakingDisabledWeightKey        = "staking-disabled-weight"
	maxNonStakerPendingMsgsKey      = "max-non-staker-pending-msgs"
	stakerMsgReservedKey            = "staker-msg-reserved"
	stakerCPUReservedKey            = "staker-cpu-reserved"
	maxPendingMsgsKey               = "max-pending-msgs"
	networkInitialTimeoutKey        = "network-initial-timeout"
	networkMinimumTimeoutKey        = "network-minimum-timeout"
	networkMaximumTimeoutKey        = "network-maximum-timeout"
	networkTimeoutHalflifeKey       = "network-timeout-halflife"
	networkTimeoutCoefficientKey    = "network-timeout-coefficient"
	sendQueueSizeKey                = "send-queue-size"
	benchlistFailThresholdKey       = "benchlist-fail-threshold"
	benchlistPeerSummaryEnabledKey  = "benchlist-peer-summary-enabled"
	benchlistDurationKey            = "benchlist-duration"
	benchlistMinFailingDurationKey  = "benchlist-min-failing-duration"
	pluginDirKey                    = "plugin-dir"
	logsDirKey                      = "log-dir"
	logLevelKey                     = "log-level"
	logDisplayLevelKey              = "log-display-level"
	logDisplayHighlightKey          = "log-display-highlight"
	snowSampleSizeKey               = "snow-sample-size"
	snowQuorumSizeKey               = "snow-quorum-size"
	snowVirtuousCommitThresholdKey  = "snow-virtuous-commit-threshold"
	snowRogueCommitThresholdKey     = "snow-rogue-commit-threshold"
	snowAvalancheNumParentsKey      = "snow-avalanche-num-parents"
	snowAvalancheBatchSizeKey       = "snow-avalanche-batch-size"
	snowConcurrentRepollsKey        = "snow-concurrent-repolls"
	snowOptimalProcessingKey        = "snow-optimal-processing"
	snowEpochFirstTransition        = "snow-epoch-first-transition"
	snowEpochDuration               = "snow-epoch-duration"
	whitelistedSubnetsKey           = "whitelisted-subnets"
	adminAPIEnabledKey              = "api-admin-enabled"
	infoAPIEnabledKey               = "api-info-enabled"
	keystoreAPIEnabledKey           = "api-keystore-enabled"
	metricsAPIEnabledKey            = "api-metrics-enabled"
	healthAPIEnabledKey             = "api-health-enabled"
	ipcAPIEnabledKey                = "api-ipcs-enabled"
	xputServerPortKey               = "xput-server-port"
	xputServerEnabledKey            = "xput-server-enabled"
	ipcsChainIDsKey                 = "ipcs-chain-ids"
	ipcsPathKey                     = "ipcs-path"
	consensusGossipFrequencyKey     = "consensus-gossip-frequency"
	consensusShutdownTimeoutKey     = "consensus-shutdown-timeout"
	fdLimitKey                      = "fd-limit"
	corethConfigKey                 = "coreth-config"
	disconnectedCheckFreqKey        = "disconnected-check-frequency"
	disconnectedRestartTimeoutKey   = "disconnected-restart-timeout"
	restartOnDisconnectedKey        = "restart-on-disconnected"
	retryBootstrap                  = "retry-bootstrap"
	retryBootstrapMaxAttempts       = "retry-bootstrap-max-attempts"
)
