package model

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"time"
)

// GetAddressLogs    总余额的账变model
type GetAddressLogs struct {
	ID       int `gorm:"primaryKey;comment:'主键'"`
	Username string
	Address  string //地址
	Status   int    // 1正常日志 2异常日志
	Created  int64  //创建时间
}

func CheckIsExistGetAddressLogs(db *gorm.DB) {
	if db.HasTable(&GetAddressLogs{}) {
		fmt.Println("数据库已经存在了!")
		db.AutoMigrate(&GetAddressLogs{})
	} else {
		fmt.Println("数据不存在,所以我要先创建数据库")
		err := db.CreateTable(&GetAddressLogs{}).Error
		if err == nil {
			fmt.Println("数据库已经存在了!")
		}
	}
}

func (l *GetAddressLogs) CreateGetAddressLogs(db *gorm.DB) {
	l.Created = time.Now().Unix()
	db.Save(l)
}
