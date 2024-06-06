package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"wzj_sign/db"
	"wzj_sign/mail"
	"wzj_sign/model"
	"wzj_sign/qr"

	"github.com/spf13/viper"
)

var getAllSignsUrl = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student/active_signs"
var signInUrl = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student-sign-in"

// 获取数据库中所有登记的OpenId
// GetAllMatchedKeys

// 获取每一个OpenId的全部签到
func GetAllSigns(openId string) ([]model.SignData, error) {
	req, err := http.NewRequest("GET", getAllSignsUrl, nil)
	if err != nil {
		log.Println("Error creating GetAllSigns request:", err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Set("Openid", openId)
	req.Header.Set("Host", "v18.teachermate.cn")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
        log.Println("Error sending GetAllSigns request:", err)
        return nil, err
    }
    defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	log.Println(openId + ":GetAllSigns Response:", string(body))
	if string(body) == `{"message":"登录信息失效，请退出后重试"}` {
		// 1s后过期
		result := db.RedisExpire("wzj:user:" + openId, 1 * time.Second)
		log.Println(openId + ":Invalid OpenId!")
		if result.Err() != nil {
			log.Println("Error setting key:", result.Err())
			return nil, result.Err()
		}
		return nil, errors.New("无效OpenId")
	}
	var signList []model.SignData
	json.Unmarshal(body, &signList)
	return signList, nil
}

// 提交签到
func Signin(sign model.SignData, openId string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
    // 生成0到10之间的随机整数
    randomNum := r.Intn(1001) // [0-1000]

	courseId := sign.CourseID
	signId := sign.SignID
	courseName := sign.Name

	// 避免重复签到
	openidSign := fmt.Sprintf("wzj:repeat:%s%d", openId, signId)
	for _, item := range db.RedisGetAllMatchedKeys("wzj:repeat:*"){
		if openidSign == item{
			log.Println(randomNum, "Repeated Sign", openId, signId)
			return
		}
	}

	if sign.IsQR != 0 {
		serverAddress := viper.GetString("app.url")
		if envAddress := os.Getenv("SERVER_ADDRESS"); envAddress != "" {
			serverAddress = envAddress
		}
		mail_title := courseName + "正在二维码签到，需要手动完成"
		mail_content := "立刻前往下方二维码网址，使用微信扫一扫完成签到，签到完成后之前提交的OpenID可能会立刻失效，如果需要再次监控需要重新添加新的OpenID到监控池。二维码捕获：" + serverAddress + "/home/qr.html?sign=" + fmt.Sprint(signId)
		go qr.InitQrSign(courseId, signId)
		mail.SendEmail(mail_title, mail_content, FindEmailByOpenId(openId))
		CoolDownFor5Min(openId, signId)
	}

	if (sign.IsGPS == 1 || ((sign.IsGPS + sign.IsQR) == 0)) {
		delay_time := viper.GetInt("app.normal_delay")
		log.Println(randomNum, "delay for", delay_time)
		time.Sleep(time.Duration(delay_time) * time.Second)
	}

	// 避免重复签到x2
	for _, item := range db.RedisGetAllMatchedKeys("wzj:repeat:*"){
		if openidSign == item{
			log.Println(randomNum, "Repeated Sign x2", openId, signId)
			return
		}
	}
	post_data := `{"courseId":` + strconv.Itoa(courseId) + `,"signId":` + strconv.Itoa(signId) + `}`
	post_data_with_gps := `{"courseId":` + strconv.Itoa(courseId) + `,"signId":` + strconv.Itoa(signId) + `,"lon":117.142737,"lat":34.212723}`
	
	// 构造普通签到和GPS签到，但其实GPS签到发送普通签到的载荷也能成功
	var (
		req *http.Request
		err error
	)
	if sign.IsGPS == 1 {
		data := strings.NewReader(post_data_with_gps)
		req, err = http.NewRequest("POST", signInUrl, data)
	}else{
		data := strings.NewReader(post_data)
		req, err = http.NewRequest("POST", signInUrl, data)
	}
	
	if err != nil {
		log.Println(randomNum, "Error creating Signin request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Set("Openid", openId)
	req.Header.Set("Host", "v18.teachermate.cn")
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
        log.Println("Error sending Signin request:", err)
        return
    }
    defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	log.Println(randomNum, "Response:", string(body))

	if strings.Contains(string(body), "你已经签到成功") {
		CoolDownFor5Min(openId, signId)
	}

	if strings.Contains(string(body), "studentRank") {
		var signResult model.SignResultData
		json.Unmarshal(body, &signResult)
		mail_title := courseName + "刚刚签到！"
		mail_content := fmt.Sprintf("【签到No.%d】你是第%d个签到的！该消息仅供参考，签到结果以实际为准。[%s/C%d/S%d/%s]", signResult.SignRank, signResult.StudentRank, courseName, courseId, signId, openId)
		mail.SendEmail(mail_title, mail_content, FindEmailByOpenId(openId))
	}
}

func FindEmailByOpenId(openid string) string {
	email, err := db.RedisGet("wzj:user:" + openid).Result()
	if err != nil {
		log.Println("Error getting value for key:", err)
		return ""
	}
	return email
}

// 设置重复签到，五分钟冷却时间
func CoolDownFor5Min(openId string, signId int){
	openidSign := fmt.Sprintf("wzj:repeat:%s%d", openId, signId)
	result := db.RedisSet( openidSign, signId, 5 * time.Minute)
	if result.Err() != nil {
		log.Println("Error setting key:", result.Err())
		return
	}
}