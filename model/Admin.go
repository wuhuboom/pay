package model

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/wangyi/GinTemplate/tools"
)

type Admin struct {
	ID         uint `gorm:"primaryKey;comment:'主键'"`
	Token      string
	Username   string
	Password   string
	MaxPond    int     `gorm:"default:1000"` //通用池子的大小  默认值 1000
	Expiration int64   `gorm:"default:30"`   //通用池子的订单过期时间  30 分钟
	PondAmount float64 `gorm:"default:5"`    //池子的金额分解点  5U
}

func CheckIsExistModeAdmin(db *gorm.DB) {
	if db.HasTable(&Admin{}) {
		fmt.Println("数据库已经存在了!")
		db.AutoMigrate(&Admin{})
	} else {
		fmt.Println("数据不存在,所以我要先创建数据库")
		err := db.CreateTable(&Admin{}).Error
		if err == nil {
			fmt.Println("数据库已经存在了!")
			db.Save(&Admin{Username: "ace001", Password: "ace001", Token: string(tools.RandString(36))})
		}
	}
}
