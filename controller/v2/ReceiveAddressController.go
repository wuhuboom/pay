package V2

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"github.com/wangyi/GinTemplate/dao/mysql"
	"io/ioutil"
	"math"

	//token "github.com/wangyi/GinTemplate/eth"
	"github.com/wangyi/GinTemplate/model"
	"github.com/wangyi/GinTemplate/tools"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
	//"github.com/ethereum/go-ethereum/accounts/abi/bind"
	//"github.com/ethereum/go-ethereum/common"
	//"github.com/ethereum/go-ethereum/ethclient"
)

// GetReceiveAddress 获取地址管理
func GetReceiveAddress(c *gin.Context) {
	action := c.Query("action")
	if action == "GET" {
		page, _ := strconv.Atoi(c.Query("page"))
		limit, _ := strconv.Atoi(c.Query("limit"))
		role := make([]model.ReceiveAddress, 0)
		Db := mysql.DB

		if add, isE := c.GetQuery("address"); isE == true {
			Db = Db.Where("address=?", add)
		}

		if add, isE := c.GetQuery("username"); isE == true {
			Db = Db.Where("username=?", add)
		}

		if add, isE := c.GetQuery("money"); isE == true {
			Db = Db.Where("money >=?", add)
		}

		//日期条件
		if start, isExist := c.GetQuery("start_time"); isExist == true {
			if end, isExist := c.GetQuery("end_time"); isExist == true {
				Db = Db.Where("updated >= ?", start).Where("updated<=?", end)
			}
		}

		var total int
		Db.Table("receive_addresses").Count(&total)
		Db = Db.Model(&model.ReceiveAddress{}).Offset((page - 1) * limit).Limit(limit).Order("created desc")
		err := Db.Find(&role).Error
		if err != nil {
			tools.ReturnError101(c, "ERR:"+err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":  0,
			"count": total,
			"data":  role,
		})
		return
	}

	if action == "ADD" {
		result := model.CreateNewReceiveAddress(mysql.DB, viper.GetString("eth.ThreeUrl"))
		if !result {
			tools.ReturnError101(c, "添加失败")
			return
		}
		tools.ReturnError200(c, "添加成功")
		return
	}

}

// Collection 资金归集
func Collection(c *gin.Context) {
	req := make(map[string]interface{})
	req["gas"] = c.Query("gas")
	req["min"] = c.Query("min")

	if req["gas"] == "" || req["min"] == "" {
		tools.ReturnError101(c, "非法参数")
		return
	}

	if addr, isExits := c.GetQuery("addr"); isExits == true {
		if addr != "" {
			addArray := strings.Split(addr, "@")
			req["addrs"] = addArray
		}
	}

	req["ts"] = time.Now().UnixMilli()
	_, err := tools.HttpRequest(viper.GetString("eth.ThreeUrl")+"/collect", req, viper.GetString("eth.ApiKey"))
	if err != nil {
		tools.ReturnError101(c, "归集失败")
		return
	}
	tools.ReturnError200(c, "归集成功")
	return
}

// GetAllMoney 获取总余额
func GetAllMoney(c *gin.Context) {
	rec := make([]model.ReceiveAddress, 0)
	err := mysql.DB.Find(&rec).Error
	if err != nil {
		tools.ReturnError101(c, "获取失败")
		return
	}
	var sumMoney float64
	for _, v := range rec {
		sumMoney = sumMoney + v.Money
	}
	tools.ReturnError200Data(c, sumMoney, "获取成功")
	return
}
func ToDecimal(ivalue interface{}, decimals int) decimal.Decimal {
	value := new(big.Int)
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, 10)
	case *big.Int:
		value = v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}

// UpdateMoneyForAddressOnce 更新地址余额
func UpdateMoneyForAddressOnce(c *gin.Context) {
	re := make([]model.ReceiveAddress, 0)
	if Address, isE := c.GetQuery("address"); isE == true {
		tools.ReturnError200(c, "执行成功")
		one := model.ReceiveAddress{}
		mysql.DB.Where("address=?", Address).First(&one)
		re = append(re, one)
	} else {
		mysql.DB.Find(&re)
	}

	go func() {
		for _, v := range re {
			url := "https://apilist.tronscanapi.com/api/account/tokens?address=" + v.Address + "&start=0&limit=20&token=&hidden=0&show=0&sortType=0"
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("accept", "application/json")
			req.Header.Set("TRON-PRO-API-KEY", viper.GetString("app.TronApiKey"))
			res, _ := http.DefaultClient.Do(req)
			body, _ := ioutil.ReadAll(res.Body)
			//fmt.Println(res)
			fmt.Println(string(body))
			var tt2 model.Ta2
			err := json.Unmarshal(body, &tt2)
			if err != nil {
				return
			}
			var newMoney float64
			newMoney = 0
			for _, datum := range tt2.Data {
				if datum.TokenId == "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" {
					newMoney, _ = tools.ToDecimal(datum.Balance, 6).Float64()

				}
			}
			//fmt.Printf("余额:%f", tt1.Data[0].AssetInUsd)
			//usd := ToDecimal(arrayA[1], 6)
			////更新数据
			ups := make(map[string]interface{})
			ups["Money"] = newMoney
			ups["Updated"] = time.Now().Unix()
			err = mysql.DB.Model(model.ReceiveAddress{}).Where("id=?", v.ID).Update(ups).Error

			fmt.Println(newMoney)

			//调动 余额变动
			if math.Abs(newMoney-v.Money) > 1 {
				change := model.AccountChange{ChangeAmount: math.Abs(newMoney - v.Money), Kinds: 1, OriginalAmount: v.Money, NowAmount: newMoney, ReceiveAddressName: v.Username}
				change.Add(mysql.DB)

			}
			if err != nil {
				fmt.Println("更新失败")
			}

			time.Sleep(1 * time.Second)
		}
		fmt.Println("检查成功!")
	}()

	tools.ReturnError200(c, "执行成功")
	return
}

// GetAddressForLastTimeGetMoney 获取回去有多少钱没有进账的地址
func GetAddressForLastTimeGetMoney(c *gin.Context) {
	day, _ := strconv.ParseInt(c.Query("day"), 10, 64)
	timeDay := day * 86400000
	all := make([]model.ReceiveAddress, 0)

	//fmt.Println(time.Now().UnixMilli())
	//fmt.Println(timeDay)
	mysql.DB.Where("the_last_get_money_time  >  ? and the_last_get_money_time !=0 ", time.Now().Unix()*1000-timeDay).Find(&all)
	//fmt.Println(len(all))
	//最后一次转账的时间  >  今天的时间-条件时间
	//tools.ReturnError200Data(c, all, "OK")
	AllDa := ""
	for _, address := range all {
		AllDa = AllDa + address.Address + "\n"

	}
	c.Writer.Write([]byte(AllDa))
	return

}
