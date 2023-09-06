package V2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/wangyi/GinTemplate/dao/mysql"
	"github.com/wangyi/GinTemplate/model"
	"github.com/wangyi/GinTemplate/tools"
	"github.com/wangyi/GinTemplate/util"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CreatePrepaidPhoneOrders 生成订单(前端传过来了)
func CreatePrepaidPhoneOrders(c *gin.Context) {

	type T struct {
		Data string `json:"data"`
	}
	var jsonDataT T
	err := c.BindJSON(&jsonDataT)
	if err != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err1:" + err.Error())
		tools.ReturnError101(c, "err1:"+err.Error())
		return
	}
	decodeString, err1 := base64.StdEncoding.DecodeString(jsonDataT.Data)
	if err1 != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err2:" + err1.Error())
		tools.ReturnError101(c, "err2:"+err1.Error())
		return
	}
	origData, err2 := util.RsaDecrypt(decodeString)
	if err2 != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err3:" + err2.Error())

		tools.ReturnError101(c, "err3:"+err2.Error())
		return
	}
	var jsonData CreatePrepaidPhoneOrdersData
	err3 := json.Unmarshal(origData, &jsonData)
	if err3 != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err4:" + err3.Error())
		tools.ReturnError101(c, "err4:"+err3.Error())
		return
	}

	////通过用户名字获取充值地址
	////判断这个 username   库里面是否存在
	//re := model.ReceiveAddress{Username: jsonData.Username}
	//if !re.ReceiveAddressIsExits(mysql.DB) {
	//	//不存在这个用户 首先要创建这个用户
	//	re.CreateUsername(mysql.DB, viper.GetString("eth.ThreeUrl"))
	//}
	//
	////返回地址
	//err = mysql.DB.Where("username=?", jsonData.Username).First(&re).Error
	//if err != nil {
	//	zap.L().Debug("CreatePrepaidPhoneOrders err5:" + err.Error())
	//	tools.ReturnError101(c, "用户没有被创建,用户非法!")
	//	return
	//}

	//判断用户是存在充值地址
	if jsonData.Username == "" {
		tools.ReturnError101(c, "用户名不可以为空")
		return
	}

	fmt.Println("用户名:" + jsonData.Username)
	re := model.ReceiveAddress{}
	err = mysql.DB.Where("username=?", jsonData.Username).First(&re).Error
	if err != nil {
		//用户不存在
		re.Username = jsonData.Username
		zap.L().Debug("CreatePrepaidPhoneOrders   用户" + jsonData.Username + " 不存在,创建")
		re.CreateUsername(mysql.DB, viper.GetString("eth.ThreeUrl"))
		if re.Address == "" {
			tools.ReturnError101(c, "返回空的地址,稍后重试")
			zap.L().Debug("CreatePrepaidPhoneOrders   用户" + jsonData.Username + " 返回空的地址,稍后重试")
			return
		}
	}

	if strings.ToUpper(jsonData.RechargeType) != "USDT" {
		zap.L().Debug("CreatePrepaidPhoneOrders RechargeType is error")
		tools.ReturnError101(c, "RechargeType is error")
		return
	}

	// 存在就更新数据
	//生成充值订单
	//判断平台订单是否重复
	err = mysql.DB.Where("platform_order=?", jsonData.PlatformOrder).First(&model.PrepaidPhoneOrders{}).Error
	if err == nil {
		zap.L().Debug("CreatePrepaidPhoneOrders 不要重复提交")
		tools.ReturnError101(c, "不要重复提交")
		return
	}
	p := model.PrepaidPhoneOrders{PlatformOrder: jsonData.PlatformOrder, RechargeAddress: re.Address, AccountOrders: jsonData.AccountOrders, Username: jsonData.Username, RechargeType: jsonData.RechargeType, BackUrl: jsonData.BackUrl, ThreeOrder: time.Now().Format("20060102150405") + strconv.Itoa(rand.Intn(100000)), Status: 1, Created: time.Now().Unix()}
	err = mysql.DB.Save(&p).Error
	if err != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err6:" + err.Error())

		tools.ReturnError101(c, "err6:"+err.Error())
		return
	}
	aUrl := viper.GetString("eth.RechargingJumpAddress") + "?Address=" + re.Address + "&RechargeType=" + jsonData.RechargeType + "&AccountOrders=" + strconv.FormatFloat(jsonData.AccountOrders, 'f', 2, 64) + "&PlatformOrder=" + jsonData.PlatformOrder

	type ReturnData struct {
		UrlAddress string
	}

	var oo ReturnData
	oo.UrlAddress = aUrl
	data, err := json.Marshal(oo)
	if err != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err7:" + err.Error())
		tools.ReturnError101(c, "err7:"+err.Error())
		return
	}

	data, err = util.RsaEncryptForEveryOne(data)
	if err != nil {
		zap.L().Debug("CreatePrepaidPhoneOrders err8:" + err.Error())
		tools.ReturnError101(c, "err8:"+err.Error())
		return
	}

	//充值订单创建成功
	tools.ReturnError200Data(c, base64.StdEncoding.EncodeToString(data), "Ok")
	return
}

// GetPrepaidPhoneOrders 充值订单
func GetPrepaidPhoneOrders(c *gin.Context) {
	action := c.Query("action")
	if action == "GET" {
		page, _ := strconv.Atoi(c.Query("page"))
		limit, _ := strconv.Atoi(c.Query("limit"))
		role := make([]model.PrepaidPhoneOrders, 0)
		Db := mysql.DB
		var total int

		type ACC struct {
			AccountOrders    float64 `json:"account_orders"`
			AccountPractical float64 `json:"account_practical"`
		}

		var bcc ACC
		// 用户名
		if content, isExist := c.GetQuery("UserID"); isExist == true {
			Db = Db.Where("username=?", content)
		}

		//平台订单号
		if content, isExist := c.GetQuery("PlatformOrder"); isExist == true {
			Db = Db.Where("platform_order=?", content)
		}

		//三方平台订单号
		if content, isExist := c.GetQuery("ThreeOrder"); isExist == true {
			Db = Db.Where("three_order=?", content)
		}

		//充值地址
		if content, isExist := c.GetQuery("RechargeAddress"); isExist == true {
			Db = Db.Where("recharge_address=?", content)
		}

		//订单状态
		if content, isExist := c.GetQuery("Status"); isExist == true {
			Db = Db.Where("status=?", content)
		}

		//是否回调
		if content, isExist := c.GetQuery("ThreeBack"); isExist == true {
			Db = Db.Where("three_back=?", content)
		}

		//日期条件
		if start, isExist := c.GetQuery("start_time"); isExist == true {
			if end, isExist := c.GetQuery("end_time"); isExist == true {
				Db = Db.Where("successfully >= ?", start).Where("successfully<=?", end)
				mysql.DB.Raw("select sum(account_orders) as account_orders ,sum(account_practical) as  account_practical from prepaid_phone_orders where successfully  BETWEEN ? AND ?", start, end).Scan(&bcc)

			}
		}

		Db.Table("prepaid_phone_orders").Count(&total)
		Db = Db.Model(&model.PrepaidPhoneOrders{}).Offset((page - 1) * limit).Limit(limit).Order("created desc")
		err := Db.Find(&role).Error
		if err != nil {
			tools.ReturnError101(c, "ERR:"+err.Error())
			return
		}

		for k, v := range role {
			address := model.ReceiveAddress{}
			err := mysql.DB.Where("username=?", v.Username).First(&address).Error
			if err == nil {
				role[k].CollectionAddress = address.Address
			}

		}

		//AccountOrders     float64 `gorm:"type:decimal(10,2)"` //充值金额 (订单金额)
		//AccountPractical  float64 `gorm:"type:decimal(10,2)"` //充值金额(实际返回金额)
		c.JSON(http.StatusOK, gin.H{
			"code":             0,
			"count":            total,
			"data":             role,
			"accountOrders":    bcc.AccountOrders,
			"accountPractical": bcc.AccountPractical,
		})
		return
	}
}

func Getaddr(c *gin.Context) {
	//{"error":"0","message":"","result":"4564554545454545"}
	c.JSON(http.StatusOK, gin.H{
		"error":   "0",
		"message": "",
		"result":  "TW2HWaLWy9pwiRN4yLju6YKW3aQ6Fw8888",
	})
	return
}

// HandBackStatus 手动回调
func HandBackStatus(c *gin.Context) {
	id := c.Query("id")
	p := model.PrepaidPhoneOrders{}
	err := mysql.DB.Where("id=?", id).First(&p).Error
	if err != nil {
		tools.ReturnError101(c, "订单不存在")
		return
	}
	err = mysql.DB.Model(&model.PrepaidPhoneOrders{}).Where("id=?", id).Update(&model.PrepaidPhoneOrders{ThreeBack: 4}).Error
	if err != nil {
		tools.ReturnError101(c, err.Error())
		return
	}
	tools.ReturnError200(c, "修改成功")
	return

}

// PullUpTheOrder 前端订单拉起  返回地址 和三方地址
func PullUpTheOrder(c *gin.Context) {
	type T struct {
		Data string `json:"data"`
	}
	var jsonDataT T
	err := c.BindJSON(&jsonDataT)
	if err != nil {
		tools.ReturnError101(c, "err:"+err.Error())
		return
	}
	decodeString, err1 := base64.StdEncoding.DecodeString(jsonDataT.Data)
	if err1 != nil {
		tools.ReturnError101(c, "err:"+err1.Error())
		return
	}
	origData, err2 := util.RsaDecrypt(decodeString)
	if err2 != nil {
		tools.ReturnError101(c, "err:"+err2.Error())
		return
	}
	var jsonData UsernameAddress
	err3 := json.Unmarshal(origData, &jsonData)
	if err3 != nil {
		tools.ReturnError101(c, "err:"+err3.Error())
		return
	}
	//判断这个 username   库里面是否存在
	re := model.ReceiveAddress{Username: jsonData.Username}
	if !re.ReceiveAddressIsExits(mysql.DB) {
		//不存在这个用户 首先要创建这个用户
		re.CreateUsername(mysql.DB, viper.GetString("eth.ThreeUrl"))
	}
	//返回地址
	err = mysql.DB.Where("username=?", jsonData.Username).First(&re).Error
	if err != nil {
		tools.ReturnError101(c, "err:"+err.Error())
		return
	}
	ThreeOrder := time.Now().Format("20060102150405") + strconv.Itoa(rand.Intn(100000))
	p := model.PrepaidPhoneOrders{RechargeAddress: re.Address, ThreeOrder: ThreeOrder}
	_, err = p.CreatePrepaidPhoneOrders(mysql.DB)
	if err != nil {
		tools.ReturnError101(c, err.Error())
		return
	}
	//充值订单创建成功
	type One struct {
		ThreeOrder      string
		RechargeAddress string
	}
	var pp One
	pp.ThreeOrder = ThreeOrder
	pp.RechargeAddress = re.Address
	data, err := json.Marshal(pp)
	if err != nil {
		tools.ReturnError101(c, err.Error())
		return
	}
	data, err = util.RsaEncryptForEveryOne(data)
	if err != nil {
		tools.ReturnError101(c, err.Error())
		return
	}
	tools.ReturnError200Data(c, data, "OK")
	return

}
