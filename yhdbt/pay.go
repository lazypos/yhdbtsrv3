package yhdbt

import (
	"crypto/md5"
	"fmt"
	"log"
)

const (
	zfb            = "zhifubao"
	zfbwap         = "zhifubao-wap"
	weixin         = "weixin"
	fmt_pay_url    = `http://wangguan.qianyifu.com:8881/gateway/pay.asp?userid=49497&orderid=%s&money=%d&hrefurl=http://lazypos.pw:51888/pay_verify&url=http://lazypos.pw:51888/pay_verify&bankid=%s&sign=%s&ext=`
	fmt_pay_sig    = `userid=49497&orderid=%s&bankid=%s&keyvalue=nGX1MqFtet0sVAwzj7RYt5Jph4Mu5Kh1d6D0EuQx`
	fmt_return_sig = `returncode=%s&userid=49497&orderid=%s&money=%s&keyvalue=nGX1MqFtet0sVAwzj7RYt5Jph4Mu5Kh1d6D0EuQx`
)

func Pay(orderid string, money int, opt string) string {
	var sig = ""
	var url = ""
	var bankid = ""
	if opt == "1" {
		bankid = weixin
	}
	if opt == "2" {
		bankid = zfbwap
	}
	sig = fmt.Sprintf(fmt_pay_sig, orderid, bankid)
	md5sig := fmt.Sprintf(`%x`, md5.Sum([]byte(sig)))
	url = fmt.Sprintf(fmt_pay_url, orderid, money, bankid, md5sig)
	log.Println("支付", url)

	return url
}

//校验MD5
func CheckSig(code, orderid, money, md5val string) bool {
	sig := fmt.Sprintf(fmt_return_sig, code, orderid, money)
	md5sig := fmt.Sprintf(`%x`, md5.Sum([]byte(sig)))
	log.Println(`md5`, md5sig, md5val)
	return md5sig == md5val
}
