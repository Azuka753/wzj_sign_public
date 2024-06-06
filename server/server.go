package server

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"wzj_sign/db"
	"wzj_sign/model"
	"wzj_sign/service"
)

func Start() {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
    	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
   		c.Header("Pragma", "no-cache")
    	c.Header("Expires", "0")
    	c.Next()
	})
	
	r.Static("/home", "./static")
	r.POST("/register", RegisterOpenIDHandler)
	r.GET("/openids", OpenIdsHandler)
	r.GET("/qr/:signId", QRCodeHandler)
	r.GET("/serverinfo", ServerInfoHandler)
	r.GET("/notice", ServerNoticeHandler)
	r.Run()
}

func RegisterOpenIDHandler(c *gin.Context) {
	var registerOpenIdData model.RegisterOpenIdData
	if err := c.ShouldBind(&registerOpenIdData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	openId := registerOpenIdData.OpenId
	value := registerOpenIdData.Value
	result := db.RedisSet("wzj:user:" + openId, value, 4 * time.Hour)
	if result.Err() != nil {
		log.Println("Error setting key:", result.Err())
		return
	}
	_, err := service.GetAllSigns(openId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "你提供的OpenId无效，请重新检查。",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "OpenId添加到监控池成功!",
		})
	}

}

func OpenIdsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"openIds": db.RedisGetAllMatchedKeys("wzj:user:*"),
	})
}

func QRCodeHandler(c *gin.Context) {
	signId := c.Param("signId")
	qrUrl, err := db.RedisGet("wzj:qr:" + signId).Result()
	if err != nil {
		log.Println("Error getting value for key:", err)
		c.JSON(http.StatusOK, gin.H{
			"qrUrl": "",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"qrUrl": qrUrl,
	})
}

func ServerInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"interval": viper.GetInt("app.interval"),
		"delay": viper.GetInt("app.normal_delay"),
	})
}

func ServerNoticeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"notice": "服务器查询签到间隔已调整至8秒，二维码签到无延迟，普通签到20秒延迟!",
	})
}
