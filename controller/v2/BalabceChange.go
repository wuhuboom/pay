package V2

import (
	"github.com/gin-gonic/gin"
	"github.com/wangyi/GinTemplate/dao/mysql"
	"github.com/wangyi/GinTemplate/model"
	"github.com/wangyi/GinTemplate/tools"
	"net/http"
	"strconv"
)

func GetBalanceChange(c *gin.Context) {
	action := c.Query("action")
	if action == "GET" {
		page, _ := strconv.Atoi(c.Query("page"))
		limit, _ := strconv.Atoi(c.Query("limit"))
		role := make([]model.BalanceChange, 0)
		Db := mysql.DB
		var total int
		//时间查询
		if created, isE := c.GetQuery("start_time"); isE == true {
			if end, isE := c.GetQuery("end_time"); isE == true {
				Db = Db.Where("created  >= ?", created).Where("created <=  ?", end)
			}
		}
		Db.Table("balance_changes").Count(&total)
		Db = Db.Model(&model.BalanceChange{}).Offset((page - 1) * limit).Limit(limit).Order("created desc")
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
}
