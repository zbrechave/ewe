package main

//二维码登录的服务端模拟
/*
获取index页面的时候生成token
二维码中带有token，websocket 中也有token
手机登录请求来的时候，取出token中的 websocket, 发出成功请求
token超时处理
token删除
*/

import (
	"github.com/chenweivip/ewe/utils"
	"github.com/skip2/go-qrcode"
	"golang.org/x/net/websocket"

	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"time"
)

const (
	PORT = ":7777"
	LETTERBYTES = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

type Token struct {
	Name string
}

type LoginReq struct {
	Status int
	Conn *websocket.Conn
}

var tokenarr = make(map[string]LoginReq) //map 一个token对象 token{status, sockconn}


func gentoken() (token string){
	b := make([]byte, 16)
	for i := range b {
		b[i] = LETTERBYTES[rand.Intn(len(LETTERBYTES))]
	}
	tokenarr[string(b)] = LoginReq{Status: 1}
	return string(b)
}


func showhandler(w http.ResponseWriter, r *http.Request) {
	//生成随机token，生成二维码， 返回给客户端
	var png []byte

	err := r.ParseForm()
	if err != nil {fmt.Printf("Error: %s", err.Error())}

	token := r.Form["token"][0]
	w.Header().Set("Content-Type", "image/png")

	ip, err := utils.ExternalIP()
	if err != nil {fmt.Printf("Error: %s", err.Error())}

	png, err = qrcode.Encode("http://" + ip.String() + PORT + "/tokenlogin?token=" + token , qrcode.Medium, 256)

	if err != nil {
		fmt.Printf("Error: %s", err.Error())} else {
		fmt.Printf("二维码 is %d bytes long\n", len(png))}
	_, _ = w.Write(png)
}


func tokenlogin(w http.ResponseWriter, r *http.Request) {
	//手机端扫描二维码 token上传 假设手机端已经登陆
	_ = r.ParseForm()
	token := r.Form["token"][0]
	_, _ = fmt.Fprintf(w, "您登陆的token是 %s ", token)

	// map value 是struct时候怎么判断呢
	if tokenarr[token].Conn != nil {
		ws := tokenarr[token].Conn
		_ = websocket.Message.Send(ws, "恭喜，登录成功")
		delete(tokenarr, token)
	}else{
		_, _ = fmt.Fprintf(w, "非法token，登录失败")
	}
}


func qrpolling(ws *websocket.Conn){
	var err error

	r := ws.Request()
	_ = r.ParseForm()
	token := r.Form["token"][0]
	fmt.Println("polling token is ", token)

	if tokenarr[token].Status != 1 {
		_ = websocket.Message.Send(ws, "非法token或者二维码已经过期")
	}

	tokenarr[token] = LoginReq{Conn: ws}

	for {
		var reply string
		if err = websocket.Message.Receive(ws, &reply); err != nil {
			fmt.Println("Can't receive")
			break
		}

		fmt.Println("websocket from client: " + reply)
		msg := "Received:  " + reply
		fmt.Println("Sending to client: " + msg)

		if err = websocket.Message.Send(ws, msg); err != nil {
			fmt.Println("Can't send")
			break
		}
	}
}


func codeTimeout(token string) {
	time.Sleep(30*time.Second)
	delete(tokenarr, token)
	fmt.Println(token, " expired")
}


func indexhandler(w http.ResponseWriter, r *http.Request) {
	var token = gentoken()
	t, _ := template.ParseFiles("templates/index.html")
	_ = t.Execute(w, Token{Name: token})
	go codeTimeout(token)
}


func main() {
	http.HandleFunc("/", indexhandler)
	http.HandleFunc("/qrcode", showhandler)
	http.HandleFunc("/tokenlogin", tokenlogin)
	http.Handle("/websocket", websocket.Handler(qrpolling))
	_ = http.ListenAndServe(PORT, nil)
}
