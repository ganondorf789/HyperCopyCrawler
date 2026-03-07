package consts

// -------- FollowModel 跟单模式（copy_trade_config.follow_model）--------

const (
	// FollowModelAssetProportional 资产等比：根据目标使用了多少本金比例，
	// 按相同比例使用跟单钱包资金下单。
	// FollowModelValue 为跟单比例百分比：
	//   100 = 与目标一致（目标用 10% 资产 → 跟单也用 10%）
	//    50 = 一半力度（目标用 10% 资产 → 跟单用 5%）
	FollowModelAssetProportional = 1

	// FollowModelPositionProportional 仓位等比：忽略本金差异，直接跟随目标仓位变化金额下单。
	// FollowModelValue 为跟单比例百分比：
	//   100 = 与目标一致（目标开仓 $100 → 跟单 $100）
	//    50 = 一半力度（目标开仓 $100 → 跟单 $50）
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
