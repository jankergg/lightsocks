package local

import (
	"net"
	"log"
	"time"
	"github.com/gwuhaolin/lightsocks/core"
)

type LsLocal struct {
	*core.SecureSocket
	running     bool
	AfterListen func(listenAddr net.Addr)
}

//新建一个本地端
//本地端的职责是:
//0.监听来自本地浏览器的代理请求
//1.转发前加密数据
//2.转发socket数据到服务端
//3.把服务端返回的数据转发给用户的浏览器
func New(encodePassword *core.Password, localAddr, serverAddr *net.TCPAddr) *LsLocal {
	return &LsLocal{
		SecureSocket: &core.SecureSocket{
			Cipher:     core.NewCipher(encodePassword),
			LocalAddr:  localAddr,
			ServerAddr: serverAddr,
		},
	}
}

//本地端启动监听给用户的浏览器调用
func (local *LsLocal) Listen() error {
	listener, err := net.ListenTCP("tcp", local.LocalAddr)
	if err != nil {
		return err
	}

	defer listener.Close()
	local.running = true

	if local.AfterListen != nil {
		local.AfterListen(listener.Addr())
	}

	for local.running {
		userConn, err := listener.AcceptTCP()
		if err != nil {
			continue
		}
		//userConn被关闭时直接清除所有数据 不管没有发送的数据
		userConn.SetLinger(0)
		go local.handleConn(userConn)
	}
	return nil
}

//停止运行当前服务端并且释放对应资源
func (local *LsLocal) Close() {
	//TODO 释放所有资源
	local.running = false
	local.SecureSocket = nil
}

func (local *LsLocal) handleConn(userConn *net.TCPConn) {
	defer userConn.Close()
	server, err := local.DialServer()
	if err != nil {
		log.Println(err)
		return
	}
	defer server.Close()
	server.SetLinger(0)
	server.SetDeadline(time.Now().Add(core.TIMEOUT))
	//进行转发
	go local.EncodeCopy(server, userConn)
	local.DecodeCopy(userConn, server)
}
