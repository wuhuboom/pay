package V2

import (
	"github.com/gin-gonic/gin"
	"github.com/wangyi/GinTemplate/dao/mysql"
	"github.com/wangyi/GinTemplate/dao/redis"
	"github.com/wangyi/GinTemplate/model"
	"github.com/wangyi/GinTemplate/tools"
	"strconv"
	"time"
)

// Login   管理员登录接口
func Login(c *gin.Context) {
	admin := model.Admin{}
	err := mysql.DB.Where("username=?", c.Query("username")).
		Where("password=?", c.Query("password")).
		First(&admin).Error
	if err != nil {
		tools.ReturnError101(c, "fail")
		return
	}
	redis.Rdb.Set("AdminToken_"+admin.Token, admin.Username, 24*time.Hour)
	tools.ReturnError200Data(c, admin, "success")
	return

}

// SetConfig 设置配置
func SetConfig(c *gin.Context) {
	//MaxPond    int     `gorm:"default:1000"`            //通用池子的大小  默认值 1000
	//Expiration int     `gorm:"default:30"`              //通用池子的订单过期时间  30 分钟
	//PondAmount float64 `gorm:"decimal(10,2),default:5"` //池子的金额分解点  5 U
	maxPond, _ := strconv.Atoi(c.Query("max_pond"))
	expiration, _ := strconv.ParseInt(c.Query("expiration"), 10, 64)
	pondAmount, _ := strconv.ParseFloat(c.Query("pond_amount"), 64)
	if maxPond == 0 {
		maxPond = 1000
	}
	if expiration == 0 {
		expiration = 30
	}
	if pondAmount == 0 {
		pondAmount = 5
	}
	//池设置要单独处理下
	var total int64
	mysql.DB.Model(&model.ReceiveAddress{}).Where("kinds=? and status=?", 2, 1).Count(&total)
	// 现在所有的地址大于池地址
	if total > int64(maxPond) {
		race := make([]model.ReceiveAddress, 0)
		mysql.DB.Where("kinds=? and status=?", 2, 1).
			Order("receive_nums asc ").Limit(total - int64(maxPond)).Find(&race)
		for _, address := range race {
			mysql.DB.Model(&model.ReceiveAddress{}).Where("id=?", address.ID).Updates(&model.ReceiveAddress{Status: 2})
		}
	} else {
		race := make([]model.ReceiveAddress, 0)
		mysql.DB.Where("kinds=? and status=?", 2, 2).
			Order("receive_nums asc ").Limit(total - int64(maxPond)).Find(&race)
		for _, address := range race {
			mysql.DB.Model(&model.ReceiveAddress{}).Where("id=?", address.ID).Updates(&model.ReceiveAddress{Status: 1})
		}
	}

	mysql.DB.Model(&model.Admin{}).Where("id=?", 1).Updates(&model.Admin{
		MaxPond:    maxPond,
		Expiration: expiration,
		PondAmount: pondAmount,
	})
	tools.ReturnError200(c, "OK")
	return
}

// GetConfig 获取配置
func GetConfig(c *gin.Context) {
	admin := model.Admin{}
	mysql.DB.Where("id=?", 1).First(&admin)
	tools.ReturnError200Data(c, admin, "OK")
	return
}
