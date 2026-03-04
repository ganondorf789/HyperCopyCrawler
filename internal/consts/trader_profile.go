package consts

// -------- AccountTotalValue 账户总价值 --------

const (
	AccountTotalValueSmall  = "small"  // 小资金：账户总价值（含现货）低于 $100,000
	AccountTotalValueMedium = "medium" // 中等资金：账户总价值（含现货）在 $100,000 - $500,000
	AccountTotalValueWhale  = "whale"  // 巨鲸：账户总价值（含现货）超过 $500,000
)

// -------- TradingRhythm 交易节奏 --------

const (
	TradingRhythmLongTerm  = "long_term"   // 长线：平均持仓时间 > 7 天
	TradingRhythmSwing     = "swing"       // 波段：1 天 < 平均持仓时间 ≤ 7 天
	TradingRhythmShortTerm = "short_term"  // 短线：1 小时 ≤ 平均持仓时间 ≤ 1 天
	TradingRhythmScalping  = "scalping"    // 超短线：平均持仓时间 < 1 小时
)

// -------- ProfitStatus 盈利状态 --------

const (
	ProfitStatusConsistent = "consistent" // 持续盈利：90天活跃≥30 且 笔数≥30, 总收益率>5% 且 盈亏比≥1.3, 期望值>0 且 近30天滚动收益>0
	ProfitStatusVolatile   = "volatile"   // 波动盈利：90天活跃≥30 且 笔数≥30, 总收益率>5%, 盈亏比<1.3 或 近30天滚动收益≤0
	ProfitStatusBreakeven  = "breakeven"  // 盈亏平衡：-5% ≤ 总收益率 ≤ +5%
)

// -------- DirectionPreference 方向偏好 --------

const (
	DirectionPreferenceBearish = "bearish" // 偏空头：多头交易比例 < 30%
	DirectionPreferenceNeutral = "neutral" // 中性：30% ≤ 多头交易比例 ≤ 70%
	DirectionPreferenceBullish = "bullish" // 偏多头：多头交易比例 > 70%
)

// -------- TradingStyle 交易风格（可多选） --------

const (
	TradingStyleHFSteady       = "hf_steady"        // 高频稳健：持仓≤1天 · 胜率>60% · 风险收益比≥1.2 · 盈亏比≥1.2 · 近30天交易≥20笔
	TradingStyleHFAggressive   = "hf_aggressive"    // 高频激进：持仓≤1天 · 胜率<50% · 风险收益比≥5 · 盈亏比≥1.5 · 近30天交易≥20笔
	TradingStyleLFSteady       = "lf_steady"        // 低频稳健：持仓>7天 · 胜率>60% · 风险收益比≥1.2 · 盈亏比≥1.2 · 近30天交易≤10笔
	TradingStyleStableProfit   = "stable_profit"    // 稳定盈利：胜率≥60% · 风险收益比≥1.5 · 最大回撤≤25% · 盈亏比≥1.5 · 夏普比率≥1.0
	TradingStyleHighRiskReward = "high_risk_reward" // 高风险高回报：风险收益比≥3 · 平均单笔盈利≥$10,000 · 最大回撤≥30%
	TradingStyleAsymmetric     = "asymmetric"       // 非对称高手：胜率<50% · 风险收益比≥5 · PF≥1.5 · 近30天交易≤20笔
	TradingStyleLowDrawdown    = "low_drawdown"     // 低回撤：过去90天整体账户最大回撤≤20%（去除出入金影响）
	TradingStyleVolatility     = "volatility"       // 波动策略：夏普比率<0.8 且 盈亏比≥1.0
)

// -------- ProfitScale 盈利规模 --------

const (
	ProfitScaleSmall  = "small"  // 小额盈利：平均单笔盈利 < $3,000
	ProfitScaleMedium = "medium" // 中等盈利：$3,000 ≤ 平均单笔盈利 ≤ $50,000
	ProfitScaleLarge  = "large"  // 大额盈利：平均单笔盈利 > $50,000
)
