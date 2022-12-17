package model

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"time"
)

// BalanceChange   总余额的账变model
type BalanceChange struct {
	ID             int     `gorm:"primaryKey;comment:'主键'"`
	OriginalAmount float64 `gorm:"type:decimal(10,2)"` // 原始金额
	ChangeAmount   float64 `gorm:"type:decimal(10,2)"` // 原始金额
	NowAmount      float64 `gorm:"type:decimal(10,2)"` //现在的金额
	Created        int64   //创建时间
}

func CheckIsExistBalanceChange(db *gorm.DB) {
	if db.HasTable(&BalanceChange{}) {

		//change := BalanceChange{OriginalAmount: 0, ChangeAmount: 10, NowAmount: 0}
		//change.Add(db)

		fmt.Println("数据库已经存在了!")
		db.AutoMigrate(&BalanceChange{})
	} else {
		fmt.Println("数据不存在,所以我要先创建数据库")
		db.CreateTable(&BalanceChange{})
		//初始化  需要获取总的余额
		type Data struct {
			Money float64 `json:"money"`
		}
		var data Data
		err := db.Raw("select sum(money) as money from receive_addresses ").Scan(&data).Error
		if err == nil {
			//新建第一条数据
			change := BalanceChange{OriginalAmount: 0, ChangeAmount: 0, NowAmount: data.Money}
			change.Add(db)
		}
	}
}

//Add   创建总账变
func (ba *BalanceChange) Add(db *gorm.DB) {
	ba.Created = time.Now().Unix()
	//获取原始的金额
	last := BalanceChange{}
	if ba.NowAmount == 0 {
		db.Last(&last)
		ba.OriginalAmount = last.NowAmount
		ba.NowAmount = last.NowAmount + ba.ChangeAmount
	}
	err := db.Save(ba).Error
	if err != nil {
		zap.L().Debug(err.Error())
		return
	}
	return
}
