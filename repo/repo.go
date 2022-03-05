package repo

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func InitDB(user, password, host, dbName string, migrate bool) (*DB, error) {
	password = url.QueryEscape(password)
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "error opening database")
	}
	db.CreateBatchSize = 300

	if migrate {
		err = db.AutoMigrate(&CheckURL{}, &ProxyData{}, &CheckURLResult{}, &TgBlockCheckerUser{})
		if err != nil {
			return nil, errors.Wrap(err, "error AutoMigrate")
		}
	}

	return &DB{db: db}, nil
}

type DB struct {
	db *gorm.DB
}

func (r *DB) AddURL(v CheckURL) error {
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&v).Error; err != nil {
		return errors.Wrap(err, "could not create element in db")
	}
	return nil
}

func (r *DB) GetListURLs() (list []CheckURL, err error) {
	err = r.db.Find(&list).Error
	return list, errors.Wrap(err, "could not get all CheckAvailableURL")
}

// func (r *DB) UpdateCheckURL_RU(v CheckURL) error {
// 	err := r.db.Model(&v).Update("IsActiveRU", "LastCheckRU").Error
// 	return errors.Wrap(err, "could not update url data RU")
// }

// func (r *DB) UpdateCheckURL_BY(v CheckURL) error {
// 	err := r.db.Model(&v).Update("IsActiveBY", "LastCheckBY").Error
// 	return errors.Wrap(err, "could not update url data BY")
// }

func (r *DB) AddProxy(v CheckURL) error {
	if err := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&v).Error; err != nil {
		return errors.Wrap(err, "could not create element in db")
	}
	return nil
}

func (r *DB) GetProxy(country string) (list []ProxyData, err error) {
	err = r.db.Where(&ProxyData{Country: country, Active: true}, "country", "active").Find(&list).Error
	return list, errors.Wrap(err, "could not get proxy list by country")
}

func (r *DB) GetCheckURLResultData(country string) (list []CheckURLResult, err error) {
	err = r.db.Preload("CheckURL").
		Where(&CheckURLResult{Country: country}, "country").
		Find(&list).Error
	return list, errors.Wrap(err, "could not get CheckURLResult")
}

func (r *DB) UpdateCheckURLResult(v CheckURLResult) error {
	err := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&v).Error
	return errors.Wrap(err, "could not save CheckURLResult")
}

func (r *DB) GetTgBlockCheckerUsersSubscribed() (list []TgBlockCheckerUser, err error) {
	err = r.db.Where(&TgBlockCheckerUser{SubscribeActive: true}, "subscribe_active").Find(&list).Error
	return list, errors.Wrap(err, "could not get TgBlockCheckerUser list")
}

func (r *DB) SetSubscribeTgBlockCheckerUser(tgChatID int64, active bool) error {
	user := TgBlockCheckerUser{
		TelegramChatID:  tgChatID,
		SubscribeActive: active,
	}

	tx := r.db.Model(&user).Where("telegram_chat_id = ?", tgChatID).Update("subscribe_active", active)
	err := tx.Error

	if errors.Is(err, gorm.ErrRecordNotFound) || tx.RowsAffected == 0 {
		err := r.db.Create(&user).Error
		if err != nil {
			return errors.Wrapf(err, "could not create model TgBlockCheckerUser, chatID:%v, active:%v", tgChatID, active)
		}
	} else if err != nil {
		return errors.Wrapf(err, "could not update model TgBlockCheckerUser, chatID:%v, active:%v", tgChatID, active)
	}

	return nil
}
