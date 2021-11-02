package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
)

func Handler(conn net.Conn) {
	buf := make([]byte, 2048)
	//读取客户端发送的内容
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileName := string(buf[:n])
	//获取客户端ip+port
	addr := conn.RemoteAddr().String()
	fmt.Println(addr + ": 客户端传输的文件名为--" + fileName)
	//告诉客户端已经接收到文件名
	conn.Write([]byte("ok"))
	//创建文件
	f, err := os.Create(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	//循环接收客户端传递的文件内容
	for {
		buf := make([]byte, 2048)
		n, _ := conn.Read(buf)
		//结束协程
		if string(buf[:n]) == "finish" {
			fmt.Println(addr + ": 协程结束")
			runtime.Goexit()
		}
		f.Write(buf[:n])
	}
	defer conn.Close()
	defer f.Close()
}

func main() {
	//创建tcp监听
	listen, err := net.Listen("tcp", ":8000")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listen.Close()

	for {
		//阻塞等待客户端
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		//创建协程
		go Handler(conn)
	}
}
