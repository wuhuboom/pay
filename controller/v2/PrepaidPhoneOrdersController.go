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
	"sync"
	"time"
)

// CreatePrepaidPhoneOrders 生成订单(前端传过来了)1
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

	//判断用户是存在充值地址
	if jsonData.Username == "" {
		tools.ReturnError101(c, "用户名不可以为空")
		return
	}
	fmt.Println("用户名:" + jsonData.Username)
	zap.L().Debug("CreatePrepaidPhoneOrders jsonData:" + string(origData))
	re := model.ReceiveAddress{}
	err = mysql.DB.Where("username=?", jsonData.Username).First(&re).Error
	if err != nil {
		//用户不存在
		re.Username = jsonData.Username
		zap.L().Debug("CreatePrepaidPhoneOrders   用户" + jsonData.Username + " 不存在,创建")
		re.CreateUsername(mysql.DB, viper.GetString("eth.ThreeUrl"))
		if re.Address == "" {
			tools.ReturnError101(c, "返回空的地址,稍后重试")
			//生成 地址日志
			logs := model.GetAddressLogs{Address: "获取地址为空,请检查大神服务器", Username: re.Username, Status: 2}
			logs.CreateGetAddressLogs(mysql.DB)
			zap.L().Debug("CreatePrepaidPhoneOrders   用户" + jsonData.Username + " 返回空的地址,稍后重试")
			return
		}
		//生成 地址日志
		logs := model.GetAddressLogs{Address: re.Address, Username: re.Username, Status: 1}
		logs.CreateGetAddressLogs(mysql.DB)
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
	p := model.PrepaidPhoneOrders{PlatformOrder: jsonData.PlatformOrder,
		RechargeAddress: re.Address, AccountOrders: jsonData.AccountOrders,
		Username: jsonData.Username, RechargeType: jsonData.RechargeType,
		BackUrl:    jsonData.BackUrl,
		ThreeOrder: time.Now().Format("20060102150405") + strconv.Itoa(rand.Intn(100000)),
		Status:     1, Created: time.Now().Unix()}
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

var GetAddressLock sync.RWMutex

// CreatePrepaidPhoneOrders2 CreatePrepaidPhoneOrders 生成订单(前端传过来了)2
func CreatePrepaidPhoneOrders2(c *gin.Context) {
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

	//判断用户是存在充值地址
	if jsonData.Username == "" {
		tools.ReturnError101(c, "用户名不可以为空")
		return
	}

	//玩家拉起的金额小于系统设置的小于系统的金额
	re := model.ReceiveAddress{}
	//判断是否有专属地址
	fmt.Println("用户名:" + jsonData.Username)
	zap.L().Debug("CreatePrepaidPhoneOrders jsonData:" + string(origData))
	err = mysql.DB.Where("username=?", jsonData.Username).First(&re).Error
	if err != nil {
		//判断这个人下注金额
		admin := model.Admin{}
		admin.PondAmount = 5
		admin.Expiration = 30
		admin.MaxPond = 1000
		mysql.DB.Where("id=?", 1).First(&admin)
		if jsonData.AccountOrders <= admin.PondAmount {
			//判断这个玩家近期是否已经拉起过地址 并且没有支付
			pp := model.PrepaidPhoneOrders{}
			err := mysql.DB.Where("username=? and created >  ? and account_orders <= ?", jsonData.Username, time.Now().Unix()-admin.Expiration*60, admin.PondAmount).First(&pp).Error
			if err == nil {
				//重复使用这个地址
				mysql.DB.Where("address=?", pp.RechargeAddress).First(&re)
				mysql.DB.Model(&model.ReceiveAddress{}).Where("address=?", pp.RechargeAddress).Updates(
					&model.ReceiveAddress{LastUseTime: time.Now().Unix() + admin.Expiration*60, ReceiveNums: re.ReceiveNums + 1})

			} else {
				GetAddressLock.Lock()
				//获取可用的地址
				err = mysql.DB.Where("kinds=? and last_use_time < ?", 2, time.Now().Unix()).Order("receive_nums asc").First(&re).Error
				//找到这个地址 err==nil  没有找到 err !=nil
				if err != nil {
					//没有找到对应的地址要去获取新的  首先判断池大小
					var pondSize int
					mysql.DB.Model(&model.ReceiveAddress{}).Where("kinds=?", 2).Count(&pondSize) //3
					if pondSize <= admin.MaxPond {
						//池子还没有满
						re.Username = "PoolConnection" + time.Now().Format("2006-01-02") + string(tools.RandString(2))
						re.Kinds = 2
						re.LastUseTime = time.Now().Unix() + admin.Expiration*60
						re.ReceiveNums = 1
						zap.L().Debug("CreatePrepaidPhoneOrders  191  用户" + jsonData.Username + " 不存在,池创建")
						re.CreateUsername(mysql.DB, viper.GetString("eth.ThreeUrl"))
						if re.Address == "" {
							tools.ReturnError101(c, "返回空的地址,稍后重试")
							//生成 地址日志
							logs := model.GetAddressLogs{Address: "获取地址为空,请检查大神服务器", Username: re.Username, Status: 2}
							logs.CreateGetAddressLogs(mysql.DB)
							zap.L().Debug("CreatePrepaidPhoneOrders   199 用户" + jsonData.Username + " 返回空的地址,稍后重试")
							GetAddressLock.Unlock()
							return
						}
						//生成 地址日志
						logs := model.GetAddressLogs{Address: re.Address, Username: jsonData.Username, Status: 1}
						logs.CreateGetAddressLogs(mysql.DB)

					} else {
						//池子已经满了
						zap.L().Debug("CreatePrepaidPhoneOrders  line 190 用户" + jsonData.Username + " 池子已经满了")
						GetAddressLock.Unlock()
						return
					}
				}
				//找到了 要更新
				mysql.DB.Model(&model.ReceiveAddress{}).Where("id=?", re.ID).Updates(
					&model.ReceiveAddress{LastUseTime: time.Now().Unix() + admin.Expiration*60, ReceiveNums: re.ReceiveNums + 1})
				GetAddressLock.Unlock()
			}
		} else {
			//金额订单大于临界点 分配专属地址
			re.Username = jsonData.Username
			zap.L().Debug("CreatePrepaidPhoneOrders   用户" + jsonData.Username + " 不存在,创建")
			re.CreateUsername(mysql.DB, viper.GetString("eth.ThreeUrl"))
			if re.Address == "" {
				tools.ReturnError101(c, "返回空的地址,稍后重试")
				//生成 地址日志
				logs := model.GetAddressLogs{Address: "获取地址为空,请检查大神服务器", Username: re.Username, Status: 2}
				logs.CreateGetAddressLogs(mysql.DB)
				zap.L().Debug("CreatePrepaidPhoneOrders   用户" + jsonData.Username + " 返回空的地址,稍后重试")
				return
			}
			//生成 地址日志
			logs := model.GetAddressLogs{Address: re.Address, Username: re.Username, Status: 1}
			logs.CreateGetAddressLogs(mysql.DB)
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
	p := model.PrepaidPhoneOrders{
		PlatformOrder:   jsonData.PlatformOrder,
		RechargeAddress: re.Address, //玩家需要充值的地址
		AccountOrders:   jsonData.AccountOrders,
		Username:        jsonData.Username,
		RechargeType:    jsonData.RechargeType,
		BackUrl:         jsonData.BackUrl,
		ThreeOrder:      time.Now().Format("20060102150405") + strconv.Itoa(rand.Intn(100000)),
		Status:          1, Created: time.Now().Unix(),
	}

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

		//for k, v := range role {
		//	address := model.ReceiveAddress{}
		//	err := mysql.DB.Where("username=?", v.Username).First(&address).Error
		//	if err == nil {
		//		role[k].CollectionAddress = address.Address
		//	}
		//
		//}

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
	if action == "UPDATED" {
		remark := c.Query("remark")
		id := c.Query("id")
		if remark == "" || id == "" {
			tools.ReturnError101(c, "remark is not  null")
			return
		}

		mysql.DB.Model(&model.PrepaidPhoneOrders{}).Where("id=?", id).Updates(&model.PrepaidPhoneOrders{Remark: remark})
		tools.ReturnError200(c, "OK")
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

// GetAddressForLastTimeGetMoney 获取回去有多少钱没有进账的地址
func GetAddressForLastTimeGetMoney(c *gin.Context) {
	day, _ := strconv.ParseInt(c.Query("day"), 10, 64)
	money := c.Query("money")
	timeDay := day * 86400000
	all := make([]model.ReceiveAddress, 0)
	//fmt.Println(time.Now().UnixMilli())
	fmt.Println(time.Now().Unix()*1000 - timeDay)
	mysql.DB.Where("the_last_get_money_time  <  ? and the_last_get_money_time !=0  and money  >= ?", time.Now().Unix()*1000-timeDay, money).Find(&all)
	//fmt.Println(len(all))
	//1693887994000
	//1693842531000
	//最后一次转账的时间  >  今天的时间-条件时间
	//tools.ReturnError200Data(c, all, "OK")
	AllDa := ""
	for _, address := range all {
		str := strconv.FormatFloat(address.Money, 'f', 2, 32)
		AllDa = AllDa + address.Address + "    " + str + "\n"

	}
	c.Writer.Write([]byte(AllDa))
	return

}
