package repo

import (
	"strings"
	"time"

	"github.com/olehbozhok/site_block_checker/proxy_util"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type CheckURL struct {
	gorm.Model
	URL string `gorm:"unique;not null"`
}

func (u *CheckURL) BeforeCreate(tx *gorm.DB) (err error) {
	u.URL = strings.TrimSpace(u.URL)
	return
}

type CheckURLResult struct {
	// gorm.Model
	CheckUrlID uint     `gorm:"primarykey;autoIncrement:false"`
	CheckURL   CheckURL `gorm:"foreignKey:CheckUrlID"`
	Country    string   `gorm:"primarykey"`
	UpdatedAt  time.Time
	IsActive   bool
	StatusCode int
	ResultData string
	ErrorText  string
}

type ProxyData struct {
	ID       uint   `gorm:"primarykey"`
	Country  string `gorm:"index"`
	Addr     string
	UserName string
	Password string
	Active   bool
	client   *proxy_util.Client
}

func (d *ProxyData) GetClient() (*proxy_util.Client, error) {
	if d.client != nil {
		return d.client, nil
	}
	client, err := proxy_util.GetClient(d.Addr, d.UserName, d.Password)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	d.client = client
	return client, nil
}

type TgBlockCheckerUser struct {
	gorm.Model
	TelegramChatID  int64 `gorm:"index,unique"`
	SubscribeActive bool
}
