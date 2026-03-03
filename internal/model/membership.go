package model

import "time"

type Membership struct {
	ID        int64      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    int64      `gorm:"not null;default:0;uniqueIndex;comment:所属用户ID" json:"user_id"`
	Level     int16      `gorm:"type:smallint;not null;default:0;comment:会员等级 0:免费 1:基础 2:高级 3:专业" json:"level"`
	StartAt   *time.Time `gorm:"comment:会员开始时间" json:"start_at"`
	ExpireAt  *time.Time `gorm:"index:idx_membership_expire_at;comment:会员到期时间" json:"expire_at"`
	Status    int16      `gorm:"type:smallint;not null;default:1;index:idx_membership_status;comment:状态 1:正常 0:禁用" json:"status"`
	CreatedAt time.Time  `gorm:"not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now();comment:更新时间" json:"updated_at"`
}

func (Membership) TableName() string {
	return "membership"
}
