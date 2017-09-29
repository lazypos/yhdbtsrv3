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
	"net/http"
	"sync"
	"time"
)

const (
	err_code_ok      = 0
	err_code_format  = 0x8001000
	err_code_exist   = 0x8001001
	err_code_noexist = 0x8001002
)

const (
	rpy_login_fmt = `{"error":"%d", "loginkey":"%s"}`
)

//用户注册服务器
type RegistServer struct {
	muxRegist sync.Mutex
}

var GRegistServer = &RegistServer{}

func (this *RegistServer) Start() error {
	http.HandleFunc("/regist", this.CRegist)
	http.HandleFunc("/login", this.CLogin)
	http.HandleFunc("/pay", this.CPayCenter)
	return http.ListenAndServe(":51888", nil)
}

//检查用户密码
func (this *RegistServer) CheckUserPass(user, pass string) int {

	if err := CheckUser(user); err != nil {
		log.Println(`[REGIST] check user error:`, err)
		return err_code_format
	}
	if err := CheckPass(pass); err != nil {
		log.Println(`[REGIST] check pass error:`, err)
		return err_code_format
	}
	return err_code_ok
}

//处理注册请求
func (this *RegistServer) CRegist(rw http.ResponseWriter, req *http.Request) {
	user := req.FormValue("user")
	pass := req.FormValue("pass")
	if code := this.CheckUserPass(user, pass); code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_format)
		return
	}

	fmt.Fprintf(rw, `{"error":"%d"}`, this.Regist(user, pass))
}

//处理充值请求
func (this *RegistServer) CPayCenter(rw http.ResponseWriter, req *http.Request) {
	player := req.FormValue("player")
	ptype := req.FormValue("paytype")
	checkid := req.FormValue("checkid")
	sessionid := req.FormValue("cookies")

	payNum := CheckPay(ptype)
	log.Println(player, payNum, checkid, sessionid)
}

//注册逻辑函数
func (this *RegistServer) Regist(username, pass string) int {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	// 查看账户是否存在
	ouid := GDBOpt.GetValue([]byte(username))
	if len(ouid) > 0 {
		log.Println(`[REGIST] user or pass exist.`, username)
		return err_code_exist
	}
	//生成唯一ID
	now := time.Now().String()
	key := fmt.Sprintf(`%s_%s_yhbdt_%s`, username, pass, now)
	uid := fmt.Sprintf(`%s%x`, username, md5.Sum([]byte(key)))
	log.Println(`[REGIST] new uid:`, uid)

	//保存信息 密码，uid，注册时间，昵称等
	mInfo := make(map[string][]byte)
	mInfo[username] = []byte(uid)
	mInfo[fmt.Sprintf(`%s_pass`, uid)] = []byte(pass)
	mInfo[fmt.Sprintf(`%s_regtime`, uid)] = []byte(now)
	mInfo[fmt.Sprintf(`%s_nick`, uid)] = []byte("匿名网友")
	mInfo[fmt.Sprintf(`%s_score`, uid)] = []byte("500")
	mInfo[fmt.Sprintf(`%s_win`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_lose`, uid)] = []byte("0")
	mInfo[fmt.Sprintf(`%s_run`, uid)] = []byte("0")

	if err := GDBOpt.PutBatch(mInfo); err != nil {
		log.Println(`[REGIST] save registe info error,`, err)
		return err_code_exist
	}
	return err_code_ok
}

//处理登陆请求
func (this *RegistServer) CLogin(rw http.ResponseWriter, req *http.Request) {
	user := req.FormValue("user")
	pass := req.FormValue("pass")
	if code := this.CheckUserPass(user, pass); code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, err_code_format)
		return
	}
	code, uid, loginKey := this.Login(user, pass)
	if code != err_code_ok {
		fmt.Fprintf(rw, `{"error":"%d"}`, code)
		return
	}
	GLogin.SaveLoginKey(loginKey, uid)
	fmt.Fprintf(rw, rpy_login_fmt, err_code_ok, loginKey)
}

//登陆逻辑
func (this *RegistServer) Login(username, pass string) (int, string, string) {
	this.muxRegist.Lock()
	defer this.muxRegist.Unlock()

	uid := GDBOpt.GetValue([]byte(username))
	if len(uid) == 0 {
		log.Println(`[REGIST] user not exist.`, username)
		return err_code_noexist, "", ""
	}

	loginkey := fmt.Sprintf(`%x`, md5.Sum([]byte(username+pass+time.Now().String())))
	return err_code_ok, string(uid[:]), loginkey
}
