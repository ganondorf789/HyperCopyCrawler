package consts

// -------- FollowModel 跟单模式（copy_trade_config.follow_model）--------

const (
	// FollowModelAssetProportional 资产等比：根据目标地址使用了多少本金比例，
	// 结合跟单钱包资金来下单（例如：目标用了总资金的 10%，跟单也用钱包资金的 10%）。
	// FollowModelValue 为倍率，1.0 = 同比例。
	FollowModelAssetProportional = 1

	// FollowModelPositionProportional 仓位等比：忽略本金差异，直接跟随目标地址的仓位变化比例下单。
	// FollowModelValue 为倍率，1.0 = 同等仓位。
	FollowModelPositionProportional = 2

	// FollowModelFixedValue 固定价值：每次开仓固定投入 FollowModelValue（USD），
	// 后续加仓/减仓按仓位比例执行。
	FollowModelFixedValue = 3
)

// -------- FollowType 跟单类型（copy_trade_config.follow_type）--------

const (
	FollowTypeAuto      = 1 // 自动跟单
	FollowTypeCondition = 2 // 条件跟单
	FollowTypeRealtime  = 3 // 实时跟单
)
