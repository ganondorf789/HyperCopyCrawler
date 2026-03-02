package model

import "time"

type Wallet struct {
	ID               int64     `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID           int64     `gorm:"not null;default:0;index:idx_wallet_user_id;comment:所属用户ID" json:"user_id"`
	Address          string    `gorm:"type:varchar(255);not null;default:'';comment:钱包地址" json:"address"`
	APIWalletAddress string    `gorm:"type:varchar(255);not null;default:'';comment:API Wallet Address" json:"api_wallet_address"`
	APISecretKey     string    `gorm:"type:varchar(255);not null;default:'';comment:API Secret Key" json:"-"`
	Remark           string    `gorm:"type:varchar(255);not null;default:'';comment:备注" json:"remark"`
	Status           int16     `gorm:"type:smallint;not null;default:1;index:idx_wallet_status;comment:状态 1:正常 0:禁用" json:"status"`
	CreatedAt        time.Time `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt        time.Time `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (Wallet) TableName() string {
	return "wallet"
}
