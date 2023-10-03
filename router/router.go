/**
 * @Author $
 * @Description //TODO $
 * @Date $ $
 * @Param $
 * @return $
 **/
package router

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	v2 "github.com/wangyi/GinTemplate/controller/v2"
	V3 "github.com/wangyi/GinTemplate/controller/v3"
	"github.com/wangyi/GinTemplate/dao/mysql"
	"github.com/wangyi/GinTemplate/dao/redis"
	eeor "github.com/wangyi/GinTemplate/error"
	"github.com/wangyi/GinTemplate/model"
	"github.com/wangyi/GinTemplate/tools"
	"github.com/wangyi/GinTemplate/util"
	"log"
	"net/http"
)

var (
	VerifyErrCode       = 1002  //校验错误Code
	ErrReturnCode       = -101  //全局返回错误的信息
	SuccessReturnCode   = 2000  //全局返回正常的数据
	IpLimitWaring       = -102  //全局限制警告
	IllegalityCode      = -103  //全局非法请求警告,一般是没有token
	NeedGoogleBind      = -104  //管理员登录需要绑定谷歌验证码
	TokenExpire         = -105  //管理员或者用户的token过期
	NoHavePermission    = -106  //没有权限
	MysqlErr            = -107  //数据的一些报错
	TaskClearing        = 300   //任务结算
	ReturnOldOrderCode  = 20001 //返回已经获取的账单
	NoBank              = 400
	SystemMinWithdrawal = 401
	NoEnoughMoney       = -888
)

func Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(Cors())
	r.Use(eeor.ErrHandler())
	r.NoMethod(eeor.HandleNotFound)
	r.NoRoute(eeor.HandleNotFound)
	r.Static("/static", "./static")
	GroupV2 := r.Group("/v2")
	{
		GroupV2.POST("/getPayInformation", v2.GetPayInformationBack)
		GroupV2.GET("/getPayInformation", v2.GetPayInformation)
		//资金归集
		//GroupV2.GET("/getPayInformation", v2.GetPayInformation)
		//充值订单管理
		GroupV2.POST("/createPrepaidPhoneOrders", v2.CreatePrepaidPhoneOrders2)
		GroupV2.GET("/getPrepaidPhoneOrders", v2.GetPrepaidPhoneOrders)
		//GetReceiveAddress 地址管理
		GroupV2.GET("/getReceiveAddress", v2.GetReceiveAddress)
		//每日统计 DailyStatistics
		GroupV2.GET("/getDailyStatistics", v2.GetDailyStatistics)
		//资金归集
		GroupV2.GET("/collection", v2.Collection)
		//更新总余额
		GroupV2.GET("/getAllMoney", v2.GetAllMoney)
		//手动回调 HandBackStatus
		GroupV2.GET("/handBackStatus", v2.HandBackStatus)
		//登录 Login
		GroupV2.GET("/login", v2.Login)
		//测试接口
		GroupV2.POST("/backUrl", func(context *gin.Context) {
			type T struct {
				Code   int    `json:"Code"`
				Msg    string `json:"Msg"`
				Result struct {
					Data string `json:"Data"`
				} `json:"Result"`
			}

			var jsonOne T
			err := context.BindJSON(&jsonOne)
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			//对date 进行解密
			decodeString, err1 := base64.StdEncoding.DecodeString(jsonOne.Result.Data)
			if err1 != nil {
				tools.ReturnError101(context, "err:"+err1.Error())
				return
			}

			one, err := util.RsaDecryptForEveryOne(decodeString)
			if err != nil {
				return
			}

			type CreatePrepaidPhoneOrdersData struct {
				PlatformOrder    string
				RechargeAddress  string
				Username         string
				AccountOrders    float64 //订单充值金额
				AccountPractical float64 //  实际充值的金额
				RechargeType     string
				BackUrl          string
			}

			var jsonData CreatePrepaidPhoneOrdersData
			err3 := json.Unmarshal(one, &jsonData)
			if err3 != nil {
				return
			}

			fmt.Println(jsonData)

		})
		GroupV2.GET("/TestBackUrl", func(context *gin.Context) {
			type Create struct {
				PlatformOrder    string
				RechargeAddress  string
				Username         string
				AccountOrders    float64 //订单充值金额
				AccountPractical float64 //  实际充值的金额
				RechargeType     string
				BackUrl          string
			}

			var tt Create
			tt.PlatformOrder = "1111111111111111111111111111"
			tt.RechargeAddress = "TW2HWaLWy9pwiRN4yLju6YKW3aQ6Fw8888"
			tt.Username = "wine"
			tt.AccountOrders = 100.00
			tt.AccountPractical = 10.00
			tt.RechargeType = "USDT"
			data, err := json.Marshal(tt)
			if err != nil {
				fmt.Println(err.Error())
			}
			data, _ = util.RsaEncryptForEveryOne(data)

			util.BackUrlToPay("http://127.0.0.1:7777/v2/backUrl", base64.StdEncoding.EncodeToString(data))

			tools.ReturnError200(context, "调佣成功")

		})
		//唤醒订单   PullUpTheOrder
		//	GroupV2.POST("/PullUpTheOrder", v2.PullUpTheOrder)
		//UpdateMoneyForAddressOnce
		GroupV2.GET("/updateMoneyForAddressOnce", v2.UpdateMoneyForAddressOnce)
		//获取账变订单  GetAccountChange
		GroupV2.GET("/getAccountChange", v2.GetAccountChange)
		//获取整个账户的张变  GetBalanceChange
		GroupV2.GET("/getBalanceChange", v2.GetBalanceChange)
		//获取多少天未进账的地址  GetAddressForLastTimeGetMoney
		GroupV2.GET("/getAddressForLastTimeGetMoney", v2.GetAddressForLastTimeGetMoney)
		//设置配置  SetConfig
		GroupV2.GET("/setConfig", v2.SetConfig)
		//获取配置 GetConfig
		GroupV2.GET("/getConfig", v2.GetConfig)

	}
	GroupV3 := r.Group("/v3")
	{
		GroupV3.GET("/sandboxCallBack", V3.SandboxCallBack)
	}
	r.GET("/getaddr", v2.Getaddr)
	r.POST("/getaddr", v2.Getaddr)
	r.Run(fmt.Sprintf(":%d", viper.GetInt("app.port")))
	return r
}

// Cors 跨域设置
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin") //请求头部
		if origin != "" {
			//接收客户端发送的origin （重要！）
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			//服务器支持的所有跨域请求的方法
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
			//允许跨域设置可以返回其他子段，可以自定义字段
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session")
			// 允许浏览器（客户端）可以解析的头部 （重要）
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
			//设置缓存时间
			c.Header("Access-Control-Max-Age", "172800")
			//允许客户端传递校验信息比如 cookie (重要)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		//允许类型校验
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "ok!")
		}

		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic info is: %v", err)
			}
		}()

		c.Next()
	}
}

// PermissionToCheck 权限校验
func PermissionToCheck() gin.HandlerFunc {
	whiteUrl := []string{"/v2/login"}
	return func(c *gin.Context) {
		if !tools.IsArray(whiteUrl, c.Request.URL.Path) {
			//token  校验
			//判断是用户还是管理员
			fmt.Println(c.Request.URL.Path)
			token := c.Request.Header.Get("token")
			//用户
			if len(token) == 36 {
				//管理员
				ad := model.Admin{}
				err := mysql.DB.Where("token=?", token).First(&ad).Error
				if err != nil {
					tools.JsonWrite(c, IllegalityCode, nil, "Sorry, your request is invalid")
					c.Abort()
				}
				//判断token 是否过期?
				if redis.Rdb.Get("AdminToken_"+token).Val() == "" {
					tools.JsonWrite(c, TokenExpire, nil, "Sorry, your login has expired")
					c.Abort()
				}
				//设置who
				c.Set("who", ad)
				c.Next()
			} else {
				tools.JsonWrite(c, IllegalityCode, nil, "Sorry, your request is invalid")
				c.Abort()
			}

		}
		c.Next()
	}

}
