package main

import (
	"time"
	"wzj_sign/db"
	"wzj_sign/server"
	"wzj_sign/service"

	"github.com/spf13/viper"
)


func main() {
	db.InitRedis()
	go startTimer()
	server.Start()
}

func startTimer() {
	interval := viper.GetInt("app.interval")
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	for {
		select {
		case <-ticker.C:
			for _, openId := range db.RedisGetAllMatchedKeys("wzj:user:*") {
				openId := openId[9:]
				signList, _ := service.GetAllSigns(openId)
				for _, sign := range signList{
					go service.Signin(sign, openId)
				}
			}
		}
	}
}