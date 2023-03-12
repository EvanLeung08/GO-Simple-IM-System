package main

import (
	"net"
	"strings"
)

type User struct {
	Name       string
	IPAddr     string
	Channel    chan string
	Connection net.Conn
	server     *Server
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:       userAddr,
		IPAddr:     userAddr,
		Channel:    make(chan string),
		Connection: conn,
		server:     server,
	}
	//启动消息监听
	go user.ListenMessage()
	return user
}

func (this *User) Online() {
	//把用户加入OnlineMap
	this.server.MapLock.Lock()
	this.server.OnlineMap[this.Name] = this
	this.server.MapLock.Unlock()
	//广播用户上线通知
	this.server.Broadcast(this, "已上线！")

}

func (this *User) Offline() {
	//移除用户
	this.server.MapLock.Lock()
	this.server.OnlineMap[this.Name] = nil
	this.server.MapLock.Unlock()
	//广播当前用户下线
	this.server.Broadcast(this, "已下线")
}

func (this *User) SendMsg(msg string) {
	this.Connection.Write([]byte(msg))
}

func (this *User) processMessage(msg string) {

	if msg == "WHO" {

		for _, user := range this.server.OnlineMap {
			this.server.MapLock.Lock()
			message := "[" + user.IPAddr + "]" + user.Name + "在线!\n"
			this.SendMsg(message)
			this.server.MapLock.Unlock()
		}

	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		//判断Name是否存在
		_, ok := this.server.OnlineMap[newName]
		if ok {
			this.SendMsg("用户名已存在！\n")
		} else {
			this.server.MapLock.Lock()
			this.server.OnlineMap[newName] = this
			delete(this.server.OnlineMap, this.Name)
			this.server.MapLock.Unlock()
			this.Name = newName
			this.SendMsg("用户名[" + newName + "]已经更新成功\n")
		}

	} else if len(msg) > 4 && msg[:3] == "to|" {
		name := strings.Split(msg, "|")[1]
		//检查输入格式是否正确
		if name == "" {
			this.SendMsg("输入格式不正确，请参考格式: to|李五|xxx\n")
			return
		}

		//检查用户是否存在
		user, ok := this.server.OnlineMap[name]
		if !ok {
			this.SendMsg("用户不存在，请再次确认！\n")
			return
		}

		//检查输入消息是否正确
		message := strings.Split(msg, "|")[2]
		if message == "" {
			this.SendMsg("输入消息格式不对，请重新输入\n")
			return
		}

		user.SendMsg(message)

	} else {
		//广播通知所有用户
		this.server.Broadcast(this, msg)
	}
}

//监听当前用户channel，有消息就发给客户端
func (this *User) ListenMessage() {
	for {
		message := <-this.Channel

		this.Connection.Write([]byte(message + "\n"))

	}
}
