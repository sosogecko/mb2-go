package model

type UserSession struct {
	UserEmail       string `json:"user_email"`
	VIP             any    `json:"vip"`
	Expire          string `json:"expire"`
	ExpireSenconds  int64  `json:"expireSenconds"`
	Passwd          string `json:"passwd"`
	Port            any    `json:"port"`
	Method          string `json:"method"`
	Protocol        string `json:"protocol"`
	Pbfs            string `json:"pbfs"`
	UID             int64  `json:"uid"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	LastCheckInTime int64  `json:"last_check_in_time"`
	Connections     int64  `json:"connections"`
	CreateTime      any    `json:"create_time,omitempty"`
	IP              string `json:"ip,omitempty"`
}

type PanelUser struct {
	UID             int64   `gorm:"column:uid;primaryKey"`
	Username        string  `gorm:"column:username"`
	Email           string  `gorm:"column:email"`
	Password        string  `gorm:"column:password"`
	Passwd          string  `gorm:"column:passwd"`
	Method          string  `gorm:"column:method"`
	Protocol        string  `gorm:"column:protocol"`
	Obfs            string  `gorm:"column:obfs"`
	InvitedBy       int64   `gorm:"column:invited_by"`
	Money           float64 `gorm:"column:money"`
	Plan            string  `gorm:"column:plan"`
	TransferEnable  int64   `gorm:"column:transfer_enable"`
	InviteCodeAmt   int64   `gorm:"column:invite_code_amount"`
	Port            int     `gorm:"column:port"`
	RebateID        string  `gorm:"column:rebate_id"`
	RebateFirst     string  `gorm:"column:rebate_first"`
	RebateCharge    string  `gorm:"column:rebate_charge"`
	LastCheckInTime int64   `gorm:"column:last_check_in_time"`
	Connections     int64   `gorm:"column:connections"`
}

func (PanelUser) TableName() string { return "user" }

type Profile struct {
	UserEmail string `gorm:"column:user_email"`
	VIP       any    `gorm:"column:vip"`
	Expire    string `gorm:"column:expire"`
	Password  string `gorm:"column:password"`
	Actively  any    `gorm:"column:actively"`
}

func (Profile) TableName() string { return "gecko.user_profile" }
