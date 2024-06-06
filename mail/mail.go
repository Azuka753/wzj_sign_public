package mail

import (
	"fmt"

	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
)

func SendEmail(title string, message string, to string) {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	mailEnabled := viper.GetBool("mail.enabled")
	if !mailEnabled {
		return
	}
	mailHost := viper.GetString("mail.host")
	mailPort := viper.GetInt("mail.port")
	mailUsername := viper.GetString("mail.username")
	mailPassword := viper.GetString("mail.password")
	mailFrom := viper.GetString("mail.from")
	
	m := gomail.NewMessage()
	m.SetHeader("From", mailFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", title)
	m.SetBody("text/plain", message)

	d := gomail.NewDialer(mailHost, mailPort, mailUsername, mailPassword)

	if err := d.DialAndSend(m); err != nil {
		fmt.Println("Error sending email to", to, err)
	} else {
		fmt.Println(to, "Email sent successfully!")
	}
}
