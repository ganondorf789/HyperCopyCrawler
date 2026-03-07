package model

import (
	"time"

	"github.com/lib/pq"
)

// CopiedPosition 状态
const (
	CopiedPositionStatusNotStarted = "NOT_STARTED"
	CopiedPositionStatusFollowing  = "FOLLOWING"
	CopiedPositionStatusStopped    = "STOPPED"
	CopiedPositionStatusEnded      = "ENDED"
	CopiedPositionStatusFailed     = "FAILED"
)

// CopiedPosition 跟单持仓表（copy_trading）
//
// 基于 copyTradeConfig 配置 + trader_position 部分字段，记录每笔跟单持仓的执行状态
//
// Status 状态：NOT_STARTED=未开始 FOLLOWING=跟单中 STOPPED=已停止 ENDED=已结束 FAILED=失败
type CopiedPosition struct {
	ID int64 `gorm:"primaryKey;comment:主键ID" json:"id"`

	// ========== copyTradeConfig 全部字段 ==========
	CopyTradingID                   int64                  `gorm:"not null;default:0;index:idx_cp_copy_trading_id;comment:跟单配置ID" json:"copy_trading_id"`
	UserID                          int64                  `gorm:"not null;default:0;index:idx_cp_user_id;comment:所属用户ID" json:"user_id"`
	TargetWallet                    string                 `gorm:"type:varchar(255);not null;default:'';comment:目标钱包地址" json:"target_wallet"`
	TargetWalletPlatform            string                 `gorm:"type:varchar(64);not null;default:'';comment:目标钱包平台" json:"target_wallet_platform"`
	Remark                          string                 `gorm:"type:varchar(255);not null;default:'';comment:备注" json:"remark"`
	FollowType                      int                    `gorm:"not null;default:1;comment:跟单类型 1:自动跟单 2:条件跟单 3:实时跟单" json:"follow_type"`
	FollowOnce                      int                    `gorm:"not null;default:0;comment:是否只跟一次 0:否 1:是" json:"follow_once"`
	PositionConditions              Conditions             `gorm:"type:jsonb;default:'[]';comment:持仓筛选条件(JSON数组)" json:"position_conditions"`
	TraderConditions                Conditions             `gorm:"type:jsonb;default:'[]';comment:交易员筛选条件(JSON数组)" json:"trader_conditions"`
	TagAccountValue                 string                 `gorm:"type:varchar(32);not null;default:'';comment:账户总价值 small/medium/whale" json:"tag_account_value"`
	TagProfitScale                  string                 `gorm:"type:varchar(32);not null;default:'';comment:盈利规模 small/medium/large" json:"tag_profit_scale"`
	TagDirection                    string                 `gorm:"type:varchar(32);not null;default:'';comment:方向偏好 short/neutral/long" json:"tag_direction"`
	TagTradingRhythm                string                 `gorm:"type:varchar(32);not null;default:'';comment:交易节奏 longterm/swing/short/scalping" json:"tag_trading_rhythm"`
	TagProfitStatus                 string                 `gorm:"type:varchar(32);not null;default:'';comment:盈利状态 steady/volatile/balanced" json:"tag_profit_status"`
	TagTradingStyles                pq.StringArray         `gorm:"type:text[];default:'{}';comment:交易风格(多选)" json:"tag_trading_styles"`
	TraderMetricPeriod              string                 `gorm:"type:varchar(16);not null;default:'7d';comment:交易员指标周期 1d/7d/30d/90d/all" json:"trader_metric_period"`
	FollowMarginMode                int                    `gorm:"not null;default:1;comment:跟单保证金模式 1:逐仓 2:全仓" json:"follow_margin_mode"`
	FollowSymbol                    string                 `gorm:"type:varchar(64);not null;default:'';comment:跟单币种" json:"follow_symbol"`
	Leverage                        int                    `gorm:"not null;default:1;comment:杠杆倍数" json:"leverage"`
	MarginMode                      int                    `gorm:"not null;default:1;comment:保证金模式 1:逐仓 2:全仓" json:"margin_mode"`
	FollowModel                     int                    `gorm:"not null;default:1;comment:跟单模式 1:固定金额 2:固定比例" json:"follow_model"`
	FollowModelValue                string                 `gorm:"type:numeric(20,8);not null;default:0;comment:跟单模式值" json:"follow_model_value"`
	MinValue                        string                 `gorm:"type:numeric(20,8);not null;default:0;comment:最小下单金额" json:"min_value"`
	MaxValue                        string                 `gorm:"type:numeric(20,8);not null;default:0;comment:最大下单金额" json:"max_value"`
	MaxMarginUsage                  string                 `gorm:"type:numeric(10,4);not null;default:0;comment:最大保证金使用率" json:"max_margin_usage"`
	TpValue                         string                 `gorm:"type:numeric(10,4);not null;default:0;comment:止盈比例" json:"tp_value"`
	SlValue                         string                 `gorm:"type:numeric(10,4);not null;default:0;comment:止损比例" json:"sl_value"`
	OptReverseFollowOrder           int                    `gorm:"not null;default:0;comment:反向跟单 0:关 1:开" json:"opt_reverse_follow_order"`
	OptFollowupDecrease             int                    `gorm:"not null;default:0;comment:跟随减仓 0:关 1:开" json:"opt_followup_decrease"`
	OptFollowupIncrease             int                    `gorm:"not null;default:0;comment:跟随加仓 0:关 1:开" json:"opt_followup_increase"`
	OptForcedLiquidationProtection  int                    `gorm:"not null;default:0;comment:强平保护 0:关 1:开" json:"opt_forced_liquidation_protection"`
	OptPositionIncreaseOpening      int                    `gorm:"not null;default:0;comment:加仓开仓 0:关 1:开" json:"opt_position_increase_opening"`
	OptSlippageProtection           int                    `gorm:"not null;default:0;comment:滑点保护 0:关 1:开" json:"opt_slippage_protection"`
	SymbolListType                  string                 `gorm:"type:varchar(16);not null;default:'WHITE';comment:交易对列表类型 WHITE:白名单 BLACK:黑名单" json:"symbol_list_type"`
	SymbolList                      string                 `gorm:"type:text;not null;default:'';comment:交易对列表,逗号分隔" json:"symbol_list"`
	MainWallet                      string                 `gorm:"type:varchar(255);not null;default:'';comment:主钱包地址" json:"main_wallet"`
	MainWalletPlatform              string                 `gorm:"type:varchar(64);not null;default:'';comment:主钱包平台" json:"main_wallet_platform"`
	CopyTradingStatus               int                    `gorm:"not null;default:1;comment:跟单配置状态 0:停用 1:启用" json:"copy_trading_status"`
	CopyTradingCreatedAt            time.Time              `gorm:"comment:跟单配置创建时间" json:"copy_trading_created_at"`
	CopyTradingUpdatedAt            time.Time              `gorm:"comment:跟单配置更新时间" json:"copy_trading_updated_at"`

	// ========== trader_position 部分字段 ==========
	TraderAddress   string `gorm:"type:varchar(42);not null;default:'';index:idx_cp_trader_addr_coin;comment:交易员钱包地址" json:"trader_address"`
	TraderCoin      string `gorm:"type:varchar(20);not null;default:'';index:idx_cp_trader_addr_coin;comment:币种" json:"trader_coin"`
	TraderSzi       string `gorm:"type:numeric;not null;default:0;comment:仓位大小（正值为多头，负值为空头）" json:"trader_szi"`
	TraderLeverageType string `gorm:"type:varchar(20);not null;default:'';comment:杠杆类型（cross/isolated）" json:"trader_leverage_type"`
	TraderLeverage  int    `gorm:"not null;default:1;comment:杠杆倍数" json:"trader_leverage"`
	TraderEntryPx   string `gorm:"type:numeric;not null;default:0;comment:入场价" json:"trader_entry_px"`
	TraderPositionValue string `gorm:"type:numeric;not null;default:0;comment:持仓价值" json:"trader_position_value"`

	// ========== 状态 ==========
	Status   string `gorm:"type:varchar(32);not null;default:'NOT_STARTED';index:idx_cp_status;comment:状态 NOT_STARTED/FOLLOWING/STOPPED/ENDED/FAILED" json:"status"`
	ErrorMsg string `gorm:"type:text;not null;default:'';comment:执行失败原因" json:"error_msg"`

	CreatedAt time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (CopiedPosition) TableName() string {
	return "copy_trading"
}
