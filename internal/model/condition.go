package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Condition 通用筛选条件
// Field: position_size(仓位大小) / leverage(杠杆倍数) / entry_price(入场价) / position_value(持仓价值) / margin_used(已用保证金) / liq_price(清算价)
// Operator: < / = / > / >= / <=
type Condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type Conditions []Condition

func (c Conditions) Value() (driver.Value, error) {
	if c == nil {
		return "[]", nil
	}
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *Conditions) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	case nil:
		*c = nil
		return nil
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
}
