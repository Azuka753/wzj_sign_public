package model

type RegisterOpenIdData struct {
	OpenId string `form:"openId" binding:"required" validate:"max=32, min=32"`
	Value  string `form:"value" binding:"required"`
}