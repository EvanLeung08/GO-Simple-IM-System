package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip        string
	Port      int
	OnlineMap map[string]*User
	Message   chan string
	MapLock   sync.RWMutex
}

//对外提供一个方法创建服务器实例
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}

	return server
}

func (this *Server) Handler(conn net.Conn) {
	fmt.Println("链接成功:", conn.RemoteAddr().String())
	//创建用户实例
	user := NewUser(conn, this)

	user.Online()

	liveChan := make(chan bool)
	//启动协程监听用户输入
	go func() {
		for {
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil && err != io.EOF {
				fmt.Println("Received error :", err)
				return
			}
			if n == 0 {
				//移除用户
				user.Offline()
				return
			}

			msg := string(buf[:n-1])
			//处理消息
			user.processMessage(msg)
			//监听用户活跃状态
			liveChan <- true
		}
	}()

	//阻塞当前协程
	for {
		select {
		case <-liveChan:
		case <-time.After(300 * time.Second):
			//用户已经超时
			//准备剔除用户
			user.SendMsg("你已经被踢了\n")
			close(user.Channel)
			conn.Close()
			//退出当前Handler
			return
		}
	}

}

func (this *Server) ListenMessage() {
	//持续监听消息通道，如果收到消息则通知所有用户
	for {
		msg := <-this.Message
		this.MapLock.Lock()
		//获取所有用户实例
		for _, user := range this.OnlineMap {
			user.Channel <- msg
			err := recover()
			if err != nil {
				fmt.Println("监听用户消息过程中接受到异常", err)
			}
			continue
		}

		this.MapLock.Unlock()
	}

}

func (this *Server) Broadcast(user *User, msg string) {
	sendMsg := "[" + user.IPAddr + "]" + user.Name + ":" + msg

	this.Message <- sendMsg
}

//启动服务器
func (this *Server) Start() {
	//建立链接
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("accept error is :", err)
		return
	}
	//关闭链接
	defer listener.Close()
	//开启协程监听
	go this.ListenMessage()
	//接收数据
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Listener accepted error is :", err)
			continue
		}
		//处理信息
		go this.Handler(conn)
	}

}
