package model

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/wangyi/GinTemplate/tools"
	"go.uber.org/zap"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
	"time"
)

// ReceiveAddress 收账地址管理
type ReceiveAddress struct {
	ID             uint `gorm:"primaryKey;comment:'主键'"`
	Username       string
	ReceiveNums    int     //收款笔数
	LastGetAccount float64 `gorm:"type:decimal(10,2)"` //最后一次的入账金额
	Address        string  //收账地址
	Money          float64 `gorm:"type:decimal(10,2)"` //账户余额
	Created        int64
	Updated        int64
}

func CheckIsExistModeReceiveAddress(db *gorm.DB) {
	if db.HasTable(&ReceiveAddress{}) {
		fmt.Println("数据库已经存在了!")
		db.AutoMigrate(&ReceiveAddress{})
	} else {
		fmt.Println("数据不存在,所以我要先创建数据库")
		err := db.CreateTable(&ReceiveAddress{}).Error
		if err == nil {
			fmt.Println("数据库已经存在了!")
		}
	}
}

// ReceiveAddressIsExits 判断转账地址是否存在
func (r *ReceiveAddress) ReceiveAddressIsExits(db *gorm.DB) bool {
	err := db.Where("username=?", r.Username).First(&ReceiveAddress{}).Error
	if err != nil {
		//错误存在(没有这个用户)
		return false
	}
	return true
}

// CreateUsername 创建这个用户  获取用户收款地址
func (r *ReceiveAddress) CreateUsername(db *gorm.DB, url string) ReceiveAddress {
	r.Created = time.Now().Unix()
	r.ReceiveNums = 0
	r.LastGetAccount = 0
	//获取收账地址  url 请求  {"error":"0","message":"","result":"4564554545454545"}   //返回数据
	req := make(map[string]interface{})
	req["user"] = r.Username
	req["ts"] = time.Now().UnixMilli()

	fmt.Println(url + "/getaddr")
	resp, err := tools.HttpRequest(url+"/getaddr", req, viper.GetString("eth.ApiKey"))
	if err != nil {
		fmt.Println(err.Error())
		return ReceiveAddress{}
	}
	var dataAttr CreateUsernameData
	if err := json.Unmarshal([]byte(resp), &dataAttr); err != nil {
		fmt.Println(err)
		return ReceiveAddress{}
	}
	if dataAttr.Result != "" {
		r.Address = dataAttr.Result
		err := db.Save(&r).Error
		if err != nil {
			return ReceiveAddress{}
		}
	}
	return *r
}

// CreateUsernameData 返回的数据 json
type CreateUsernameData struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func (r *ReceiveAddress) UpdateReceiveAddressLastInformation(db *gorm.DB) bool {
	re := ReceiveAddress{}
	err := db.Where("username=?", r.Username).First(&re).Error
	if err == nil {
		nums := re.ReceiveNums + 1
		err := db.Model(&ReceiveAddress{}).Where("id=?", re.ID).Update(&ReceiveAddress{ReceiveNums: nums, LastGetAccount: r.LastGetAccount, Updated: r.Updated, Money: r.Money}).Error
		if err == nil {
			//更新账变
			change := AccountChange{ChangeAmount: math.Abs(re.Money - r.Money), Kinds: 2, OriginalAmount: re.Money, NowAmount: r.Money, ReceiveAddressName: r.Username}
			change.Add(db)
			return true
		}
	}
	return false
}

func (r *ReceiveAddress) UpdateReceiveAddressLastInformationTo0(db *gorm.DB) bool {
	re := ReceiveAddress{}
	err := db.Where("username=?", r.Username).First(&re).Error
	if err == nil {
		zap.L().Debug("余额清0,用户:" + r.Username)

		updated := make(map[string]interface{})
		updated["Updated"] = r.Updated
		updated["Money"] = 0
		err := db.Model(&ReceiveAddress{}).Where("id=?", re.ID).Update(updated).Error
		if err == nil {
			return true
		}
	}

	zap.L().Debug("余额清0,用户:" + r.Username + "没有找到")

	return false
}

// CreateNewReceiveAddress 创建新的地址
func CreateNewReceiveAddress(db *gorm.DB, url string) bool {
	//随机生成新的用户名
	username := tools.RandString(40)
	err := db.Where("username=?", string(username)).First(&ReceiveAddress{}).Error
	if err == nil {
		//找到了
		return false
	}
	r2 := ReceiveAddress{Username: string(username)}
	r2.CreateUsername(db, url)
	return true
}

func CheckTx(db *gorm.DB) {
	for true {
		rA := make([]ReceiveAddress, 0)
		db.Find(&rA)
		for _, address := range rA {
			//address.Address = "TCtFtwYAPUg2f5nQ9bzop9vpSPtg67hXpb"
			url := "https://apilist.tronscanapi.com/api/token_trc20/transfers?limit=20&start=0&sort=-timestamp&count=true&relatedAddress=" + address.Address
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				continue
			}
			req.Header.Add("accept", "application/json")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				continue
			}
			var tt1 Ta
			err = json.Unmarshal(body, &tt1)
			if err != nil {
				return
			}
			if len(tt1.TokenTransfers) > 0 {
				for _, transfer := range tt1.TokenTransfers {
					//fmt.Println(transfer.TransactionId)
					//判断这个  tx  是否存在
					if strings.ToUpper(transfer.ContractAddress) == strings.ToUpper("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t") {
						err := db.Where("tx_hash=?", transfer.TransactionId).First(&PayOrder{}).Error
						if err != nil {
							//不存在 就添加
							type PayOrder struct {
								ID          uint    `gorm:"primaryKey;comment:'主键'"`
								TxHash      string  //转账hash 值
								BlockNumber int     //区块号
								Timestamp   int64   //时间戳
								FromAddress string  //转账地址
								ToAddress   string  //收账地址
								Amount      float64 `gorm:"type:decimal(10,2)"` //金额
								Token       string  //token
								UserID      string  //用户id
								Created     int64
								Date        string
							}
							newMoney, _ := tools.ToDecimal(transfer.Quant, 6).Float64()
							rEA := ReceiveAddress{}
							err := db.Where("address=?", transfer.ToAddress).First(&rEA).Error
							if err == nil {
								order := PayOrder{Created: time.Now().Unix(), TxHash: transfer.TransactionId,
									BlockNumber: transfer.Block,
									Timestamp:   transfer.BlockTs / 1000,
									FromAddress: transfer.FromAddress,
									ToAddress:   transfer.ToAddress,
									Amount:      newMoney,
									Date:        time.Now().Format("2006-01-02"),
									Token:       "USDT",
									UserID:      rEA.Username,
								}
								db.Save(&order)
								change := BalanceChange{OriginalAmount: 0, ChangeAmount: newMoney, NowAmount: 0}
								change.Add(db)
							}

						}
					}

				}
			}

			//获取账户的余额
			url = "https://apilist.tronscanapi.com/api/account/tokens?address=" + address.Address + "&start=0&limit=20&token=&hidden=0&show=0&sortType=0"
			req, err = http.NewRequest("GET", url, nil)
			if err != nil {
				continue
			}
			req.Header.Add("accept", "application/json")
			res, _ = http.DefaultClient.Do(req)
			body, _ = ioutil.ReadAll(res.Body)
			var tt2 Ta2
			err = json.Unmarshal(body, &tt2)
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
			//更新余额
			affected := db.Model(&ReceiveAddress{}).Where("address=?", address.Address).Updates(map[string]interface{}{"Money": newMoney}).Error
			fmt.Println(affected)
			time.Sleep(5 * time.Millisecond)
		}

		time.Sleep(30 * 60 * time.Second)

	}

}

type Ta struct {
	Total        int `json:"total"`
	ContractInfo struct {
	} `json:"contractInfo"`
	RangeTotal     int `json:"rangeTotal"`
	TokenTransfers []struct {
		TransactionId  string `json:"transaction_id"`
		BlockTs        int64  `json:"block_ts"`
		FromAddress    string `json:"from_address"`
		FromAddressTag struct {
			FromAddressTag     string `json:"from_address_tag"`
			FromAddressTagLogo string `json:"from_address_tag_logo"`
		} `json:"from_address_tag"`
		ToAddress    string `json:"to_address"`
		ToAddressTag struct {
			ToAddressTagLogo string `json:"to_address_tag_logo"`
			ToAddressTag     string `json:"to_address_tag"`
		} `json:"to_address_tag"`
		Block           int    `json:"block"`
		ContractAddress string `json:"contract_address"`
		TriggerInfo     struct {
			Method    string `json:"method"`
			Data      string `json:"data"`
			Parameter struct {
				Value string `json:"_value"`
				To    string `json:"_to"`
			} `json:"parameter"`
			MethodName      string `json:"methodName"`
			ContractAddress string `json:"contract_address"`
			CallValue       int    `json:"call_value"`
		} `json:"trigger_info"`
		Quant          string `json:"quant"`
		ApprovalAmount string `json:"approval_amount"`
		EventType      string `json:"event_type"`
		ContractType   string `json:"contract_type"`
		Confirmed      bool   `json:"confirmed"`
		ContractRet    string `json:"contractRet"`
		FinalResult    string `json:"finalResult"`
		TokenInfo      struct {
			TokenId      string `json:"tokenId"`
			TokenAbbr    string `json:"tokenAbbr"`
			TokenName    string `json:"tokenName"`
			TokenDecimal int    `json:"tokenDecimal"`
			TokenCanShow int    `json:"tokenCanShow"`
			TokenType    string `json:"tokenType"`
			TokenLogo    string `json:"tokenLogo"`
			TokenLevel   string `json:"tokenLevel"`
			IssuerAddr   string `json:"issuerAddr"`
			Vip          bool   `json:"vip"`
		} `json:"tokenInfo"`
		FromAddressIsContract bool `json:"fromAddressIsContract"`
		ToAddressIsContract   bool `json:"toAddressIsContract"`
		Revert                bool `json:"revert"`
	} `json:"token_transfers"`
}

type Ta2 struct {
	Total int `json:"total"`
	Data  []struct {
		Amount           interface{} `json:"amount"`
		Quantity         interface{} `json:"quantity"`
		TokenId          string      `json:"tokenId"`
		TokenPriceInUsd  float64     `json:"tokenPriceInUsd"`
		TokenName        string      `json:"tokenName"`
		TokenAbbr        string      `json:"tokenAbbr"`
		TokenCanShow     int         `json:"tokenCanShow"`
		TokenLogo        string      `json:"tokenLogo"`
		TokenPriceInTrx  float64     `json:"tokenPriceInTrx"`
		AmountInUsd      float64     `json:"amountInUsd"`
		Balance          string      `json:"balance"`
		TokenDecimal     int         `json:"tokenDecimal"`
		TokenType        string      `json:"tokenType"`
		Vip              bool        `json:"vip"`
		NrOfTokenHolders int         `json:"nrOfTokenHolders,omitempty"`
		TransferCount    int         `json:"transferCount,omitempty"`
		Project          string      `json:"project,omitempty"`
	} `json:"data"`
	ContractMap struct {
		TR7NHqjeKQxGTCi8Q8ZY4PL8OtSzgjLj6T bool `json:"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"`
		Field2                             bool `json:"_"`
	} `json:"contractMap"`
	ContractInfo struct {
		TR7NHqjeKQxGTCi8Q8ZY4PL8OtSzgjLj6T struct {
			Tag1    string `json:"tag1"`
			Tag1Url string `json:"tag1Url"`
			Name    string `json:"name"`
			Vip     bool   `json:"vip"`
		} `json:"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"`
	} `json:"contractInfo"`
}
