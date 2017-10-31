/*=================================
User name: lazypos
Time: 2017-09-21
Explain:
=================================*/
package yhdbt

import (
	"crypto/md5"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	err_code_ok        = 0
	err_code_format    = 0x8001000
	err_code_exist     = 0x8001001
	err_code_noexist   = 0x8001002
	err_code_nickexist = 0x8001003 //昵称存在
	err_code_busy      = 0x8001004 //太频繁
	err_code_verify    = 0x8001005 //验证码错误

	err_pay_format = 20
	err_pay_login  = 21 // 充值登陆错误
	err_pay_busy   = 22
	err_pay_system = 23 //系统错误

	err_modify_noexist = 30 //手机号不存在
	err_modify_format  = 31 //手机号格式不正确
	err_modify_verify  = 32 //验证码不对
	err_modify_busy    = 33 //验证码频繁
	err_modify_system  = 34 //系统错误
)

const (
	rpy_login_fmt = `{"error":"%d", "loginkey":"%s"}`
	rpy_pay_fmt   = `{"error":"%d"}`
)

type QueryPayMsg struct {
	ch  chan bool
	uid string
	ok  bool
}

//用户注册服务器
type RegistServer struct {
	muxRegist sync.Mutex
	muxPay    sync.Mutex
	mapTime   map[string]int64  //验证码时间
	mapVeri   map[string]string //验证码
	mapOrder  map[string]string //[orderid]uid
	mapScore  map[string]int    //dingdan /fenzhi
}

var GRegistServer = &RegistServer{}

func (this *RegistServer) Start() error {
	rand.Seed(time.Now().UnixNano())
	this.mapTime = make(map[string]int64)
	this.mapVeri = make(map[string]string)
	this.mapOrder = make(map[string]string)
	this.mapScore = make(map[string]int)
	http.HandleFunc("/regist", this.CRegist)        //注册
	http.HandleFunc("/login", this.CLogin)          //登陆
	http.HandleFunc("/version", this.CVersion)      //版本查询
	http.HandleFunc("/modify", this.CModify)        //修改密码验证码
	http.HandleFunc("/password", this.CPassword)    //修改密码
	http.HandleFunc("/verify", this.CVerify)        //获取验证码
	http.HandleFunc("/pay", this.CPayCenter)        //充值
	http.HandleFunc("/pay_verify", this.CPayVerify) //充值验证
	http.Handle("/", http.FileServer(http.Dir("web")))
	return http.ListenAndServe(":51888", nil)
}

//处理充值请求
func (this *RegistServer) CPayCenter(rw http.ResponseWriter, req *http.Request) {
	user := req.FormValue("user")
	pass := req.FormValue("pass")
	pay := req.FormValue("type")
	opt := req.FormValue("opt")

	money := 0
	score := 0
	switch pay {
	case "1":
		money = 5
		score = 500
	case "2":
		money = 20
		score = 2400
	case "3":
		money = 100
		score = 14000
	case "4":
		money = 600
		score = 96000
	}
	if money == 0 {
		fmt.Fprintf(rw, rpy_pay_fmt, err_pay_format)
		return
	}
	if code := this.CheckUserPassNick(user, pass, "hello"); code != err_code_ok {
		fmt.Fprintf(rw, rpy_pay_fmt, err_pay_format)
		return
	}

	log.Println(`[pay] 用户充值`, req.RemoteAddr, user)

	code, uid, orderid := this.Login(user, pass)
	if code != err_code_ok {
		fmt.Fprintf(rw, rpy_pay_fmt, err_pay_login)
		return
	}
	log.Println("充值登陆成功")

	this.muxPay.Lock()
	defer this.muxPay.Unlock()
	this.mapOrder[orderid] = uid
	this.mapScore[orderid] = score
	//充值
	urlpay := Pay(orderid, money, opt)
	fmt.Fprintf(rw, "%s", urlpay)
}

//充值验证
func (this *RegistServer) CPayVerify(rw http.ResponseWriter, req *http.Request) {
	code := req.FormValue("returncode")
	orderid := req.FormValue("orderid")
	money := req.FormValue("money")
	sig := req.FormValue("sign")
	log.Println("收到订单回调", req.RequestURI)

	this.muxPay.Lock()
	defer this.muxPay.Unlock()
	uid, ok := this.mapOrder[orderid]
	if !ok {
		log.Println("异常订单：", orderid)
		fmt.Fprintf(rw, "success")
		return
	}
	if code == "1" && CheckSig(code, orderid, money, sig) {
		log.Println("订单成功！", orderid, money)
		if strings.Contains(money, ".00") {
			GDBOpt.PutValue([]byte(fmt.Sprintf(`%s_%s`, uid, orderid)), []byte(money))
			GHall.AddScore(this.mapScore[orderid], uid)
			fmt.Fprintf(rw, "success")
			return
		}
	}
	fmt.Fprintf(rw, "充值成功！请重新登陆游戏查看积分，如有异常请联系QQ群管理员。")
}

//获取验证码
func (this *RegistServer) CVerify(rw http.ResponseWriter, req *http.Request) {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	log.Println(`验证码,`, req.RemoteAddr)

	user := req.FormValue("phone")
	//验证手机号
	if err := CheckUser(user); err != nil {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_format)
		return
	}
	//注册总量限制
	if len(this.mapTime) > 200 {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_busy)
		return
	}

	//是否频繁
	now := time.Now().Unix()
	t, ok := this.mapTime[user]
	if ok && now-t < 180 {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_busy)
		return
	}
	this.mapTime[user] = now

	//是否已经注册过
	ouid := GDBOpt.GetValue([]byte(user))
	if len(ouid) > 0 {
		log.Println(`[REGIST] 手机号已经存在.`, user)
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_exist)
		return
	}

	//发送短信
	n := time.Now().Unix() + rand.Int63()
	veri := fmt.Sprintf("%6.d", n)[:6]
	log.Println(`[REGIST] `, user, "验证码", veri)
	if !SendSMS(user, veri) {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_busy)
		return
	}
	this.mapVeri[user] = veri
	fmt.Fprintf(rw, `{"error":"%d"}`, err_code_ok)
}

//修改密码验证码
func (this *RegistServer) CModify(rw http.ResponseWriter, req *http.Request) {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	log.Println(`修改密码验证码,`, req.RemoteAddr)

	user := req.FormValue("phone")
	//验证手机号
	if err := CheckUser(user); err != nil {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_format)
		return
	}

	//是否频繁
	now := time.Now().Unix()
	t, ok := this.mapTime[user]
	if ok && now-t < 180 {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_busy)
		return
	}
	this.mapTime[user] = now

	//是否已经注册过
	ouid := GDBOpt.GetValue([]byte(user))
	if len(ouid) == 0 {
		log.Println(`修改密码手机号不存在.`, user)
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_noexist)
		return
	}

	//发送短信
	n := time.Now().Unix() + rand.Int63()
	veri := fmt.Sprintf("%6.d", n)[:6]
	log.Println(`[REGIST] `, user, "验证码", veri)
	if !SendSMS(user, veri) {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_busy)
		return
	}
	this.mapVeri[user] = veri
	fmt.Fprintf(rw, `{"error":"%d"}`, 0)
}

//修改密码
func (this *RegistServer) CPassword(rw http.ResponseWriter, req *http.Request) {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	user := req.FormValue("user")
	pass := req.FormValue("pass")
	veri := req.FormValue("veri")
	log.Println("修改密码：", user)

	v, ok := this.mapVeri[user]
	if !ok || v != veri {
		log.Println(`[修改密码：] 验证码错误`, user)
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_verify)
		return
	}

	if code := this.CheckUserPassNick(user, pass, "hello"); code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_format)
		return
	}
	// 修改密码
	//拿UID
	uid := GDBOpt.GetValue([]byte(user))
	if len(uid) == 0 {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_noexist)
		return
	}
	//修改密码
	if err := GDBOpt.PutValue([]byte(fmt.Sprintf(`%s_pass`, uid)), []byte(pass)); err != nil {
		log.Println(`修改密码失败:`, uid, pass)
		fmt.Fprintf(rw, `{"error":"%d"}`, err_modify_system)
		return
	}

	fmt.Fprintf(rw, `{"error":"%d"}`, 0)
}

//检查用户密码
func (this *RegistServer) CheckUserPassNick(user, pass, nick string) int {

	if err := CheckUser(user); err != nil {
		log.Println(`[REGIST] check user error:`, err)
		return err_code_format
	}
	if err := CheckPass(pass); err != nil {
		log.Println(`[REGIST] check pass error:`, err)
		return err_code_format
	}
	if err := CheckNick(nick); err != nil {
		log.Println(`[REGIST] check nick error:`, err)
		return err_code_format
	}
	return err_code_ok
}

//处理注册请求
func (this *RegistServer) CRegist(rw http.ResponseWriter, req *http.Request) {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	user := req.FormValue("user")
	pass := req.FormValue("pass")
	nick := req.FormValue("nick")
	sex := req.FormValue("sex")
	veri := req.FormValue("veri")
	log.Println(`[REGIST] regist：`, nick)

	v, ok := this.mapVeri[user]
	if !ok || v != veri {
		log.Println(`[REGIST] 验证码错误`, user)
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_verify)
		return
	}

	if code := this.CheckUserPassNick(user, pass, nick); code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_format)
		return
	}
	fmt.Fprintf(rw, `{"error":"%d"}`, this.Regist(user, pass, nick, sex))
}

//注册逻辑函数
func (this *RegistServer) Regist(username, pass, nick, sex string) int {

	// 查看账户是否存在
	ouid := GDBOpt.GetValue([]byte(username))
	if len(ouid) > 0 {
		log.Println(`[REGIST] user or pass exist.`, username)
		return err_code_exist
	}
	//查看昵称是否存在
	onick := GDBOpt.GetValue([]byte(nick))
	if len(onick) > 0 {
		log.Println(`[REGIST] nick exist.`, username)
		return err_code_nickexist
	}
	//生成唯一ID
	now := time.Now().String()
	key := fmt.Sprintf(`%s_%s_yhbdt_%s`, username, pass, now)
	uid := fmt.Sprintf(`%s%x`, username, md5.Sum([]byte(key)))
	log.Println(`[REGIST] new uid:`, uid)

	//保存信息 密码，uid，注册时间，昵称等
	mInfo := make(map[string][]byte)
	mInfo[username] = []byte(uid)
	mInfo[nick] = []byte(uid)
	mInfo[fmt.Sprintf(`%s_pass`, uid)] = []byte(pass)
	mInfo[fmt.Sprintf(`%s_regtime`, uid)] = []byte(now)
	mInfo[fmt.Sprintf(`%s_nick`, uid)] = []byte(nick)
	mInfo[fmt.Sprintf(`%s_score`, uid)] = []byte("200")
	mInfo[fmt.Sprintf(`%s_win`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_lose`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_run`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_he`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_zong`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_regtime`, uid)] = []byte(time.Now().String())
	//默认男性
	if sex == "1" {
		mInfo[fmt.Sprintf(`%s_sex`, uid)] = []byte("1")
	} else {
		mInfo[fmt.Sprintf(`%s_sex`, uid)] = []byte("0")
	}

	if err := GDBOpt.PutBatch(mInfo); err != nil {
		log.Println(`[REGIST] save registe info error,`, err)
		return err_code_exist
	}
	log.Println(`[REGIST] 注册成功！`)
	return err_code_ok
}

//处理登陆请求
func (this *RegistServer) CLogin(rw http.ResponseWriter, req *http.Request) {
	user := req.FormValue("user")
	pass := req.FormValue("pass")
	if code := this.CheckUserPassNick(user, pass, "hello"); code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_format)
		return
	}
	code, uid, loginKey := this.Login(user, pass)
	if code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, code)
		return
	}
	GLogin.SaveLoginKey(loginKey, uid)
	log.Println(`[REGIST] login ok`, user, loginKey, uid)
	fmt.Fprintf(rw, rpy_login_fmt, err_code_ok, loginKey)
}

//登陆逻辑
func (this *RegistServer) Login(username, pass string) (int, string, string) {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	uid := GDBOpt.GetValue([]byte(username))
	if len(uid) == 0 {
		log.Println(`[REGIST] user not exist.`, username, pass)
		return err_code_noexist, "", ""
	}

	loginkey := fmt.Sprintf(`%d%x`, time.Now().Unix(), md5.Sum([]byte(username+pass+time.Now().String())))
	return err_code_ok, string(uid[:]), loginkey
}

func (this *RegistServer) CVersion(rw http.ResponseWriter, req *http.Request) {
	log.Println(`[REGIST] 查询版本`, req.RemoteAddr)
	fmt.Fprintf(rw, fmt_query_version)
}
