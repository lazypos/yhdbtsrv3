package yhdbt

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	//"math/rand"
	"net/http"
	"net/url"
	//"strings"
	//"time"
)

const (
	//短信发送格式
	fmt_sms = `{
    "tel": {
        "nationcode": "86",
        "mobile": "%s"
    }, 
    "type": 0, 
    "msg": "您的验证码为 %s，如非本人操作，请忽略本短信。",
    "sig": "%s",
    "time": %d,
    "extend": "",
    "ext": ""
    }`

	fmt_post_url = `https://yun.tim.qq.com/v5/tlssmssvr/sendsms?sdkappid=1400045077&random=%v`

	fmt_sig = `appkey=7f8e55912ae1b848feb40bf51ff6e7b4&random=%v&time=%v&mobile=%s`

	fmt_post_url_new = `https://sms.yunpian.com/v2/sms/single_send.json`
	fmt_sms_new      = `{
		"apikey":"ea63b2b0283fc779dd1c538094ac0154",
		"mobile":"%s",
		"text":"【大板同】您的验证码是%s。如非本人操作，请忽略本短信"
	}`
)

func CreateSig(mobile string, randnum, timenow int64) string {
	str := fmt.Sprintf(fmt_sig, randnum, timenow, mobile)
	return fmt.Sprintf(`%x`, sha256.Sum256([]byte(str)))
}

//发送验证码
func SendSMS(mobile, verification string) bool {
	//创建随机数
	//now := time.Now().Unix()
	//n := rand.Int63()
	//url := fmt.Sprintf(fmt_post_url, n)
	//sig := CreateSig(mobile, n, now)
	//url := fmt_post_url_new

	//发送内容
	//content := fmt.Sprintf(fmt_sms, mobile, verification, sig, now)
	//content := fmt.Sprintf(fmt_sms_new, mobile, verification)
	return Postquery(fmt_post_url_new, mobile, verification)
}

type SMSRecv struct {
	Rst   int    `json:"result"`
	Msg   string `json:"errmsg"`
	Code  int    `json:"code"`
	MsgEx string `json:"msg"`
}

//发送post请求
func Postquery(urladdr, mobile, verification string) bool {
	//log.Println(content)

	apikey := "ea63b2b0283fc779dd1c538094ac0154"
	text := fmt.Sprintf(`【大板同】您的验证码是%s。如非本人操作，请忽略本短信`, verification)
	data_send_sms := url.Values{"apikey": {apikey}, "mobile": {mobile}, "text": {text}}

	resp, err := http.PostForm(urladdr, data_send_sms)
	if err != nil {
		log.Println(`PostForm error`, err)
		return false
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(`ioutil.ReadAll error`, err)
		return false
	}
	log.Println(string(data[:]))
	sms := &SMSRecv{}
	if err := json.Unmarshal(data, sms); err != nil {
		log.Println(`解析json错误`, err)
		return false
	}
	if sms.Code != 0 {
		log.Println(sms.MsgEx)
	}
	return sms.Code == 0
}
