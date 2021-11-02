package main

import (
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"io"
	"os"
)

func handleError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

func main() {

	quicServerAddr := "127.0.0.1:4242"

	//mpquic server
	quicConfig := &quic.Config{
		CreatePaths: true,
	}

	sess, err := quic.DialAddr(quicServerAddr, &tls.Config{InsecureSkipVerify: true}, quicConfig)
	handleError(err)

	stream, err := sess.OpenStream()
	handleError(err)

	defer stream.Close()
	fmt.Print("请输入文件的完整路径：")
	//创建切片，用于存储输入的路径
	var str string
	fmt.Scan(&str)
	//fileInfo, err := os.Stat(str)
	//fileName := fileInfo.Name()

	//stream.Write([]byte(fileName))

	f, err := os.Open(str)
	handleError(err)
	defer f.Close()

	for {
		buf := make([]byte, 2048)
		n, err := f.Read(buf)
		if err != nil && io.EOF == err {
			fmt.Println("文件传输完成")
			//告诉服务端结束文件接收
			stream.Write([]byte("finish"))
			break
		}
		stream.Write(buf[:n])
	}
}
