package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

//发送文件到服务端
func SendFile(filePath string, fileSize int64, conn net.Conn) {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	var count int64
	for {
		buf := make([]byte, 2048)
		//读取文件内容
		n, err := f.Read(buf)
		if err != nil && io.EOF == err {
			fmt.Println("文件传输完成")
			//告诉服务端结束文件接收
			conn.Write([]byte("finish"))
			return
		}
		//发送给服务端
		conn.Write(buf[:n])

		count += int64(n)
		sendPercent := float64(count) / float64(fileSize) * 100
		value := fmt.Sprintf("%.2f", sendPercent)
		//打印上传进度
		fmt.Println("文件上传：" + value + "%")
	}
}

func main() {
	fmt.Print("请输入文件的完整路径：")
	//创建切片，用于存储输入的路径
	var str string
	fmt.Scan(&str)
	//获取文件信息
	fileInfo, err := os.Stat(str)
	if err != nil {
		fmt.Println(err)
		return
	}
	//创建客户端连接
	conn, err := net.Dial("tcp", ":8000")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	//文件名称
	fileName := fileInfo.Name()
	//文件大小
	fileSize := fileInfo.Size()
	//发送文件名称到服务端
	conn.Write([]byte(fileName))
	buf := make([]byte, 2048)
	//读取服务端内容
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	revData := string(buf[:n])
	if revData == "ok" {
		//发送文件数据
		begin := time.Now()
		SendFile(str, fileSize, conn)
		end := time.Since(begin)
		fmt.Println("throughtPut(MB) :", 50/float64(float64(end)/float64(time.Second)))
	}
}
