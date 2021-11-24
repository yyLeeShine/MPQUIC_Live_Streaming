package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	quic "github.com/lucas-clemente/quic-go"
	"math/big"
	"os"
	"time"
)

const quicServerAddr = "0.0.0.0:4242"

func HandleError(err error) {
	if err != nil {
		fmt.Println("Error: ")
		os.Exit(1)
	}
}

func main() {

	quicConfig := &quic.Config{
		CreatePaths: true,
	}

	// initializing mpquic server
	fmt.Println("Attaching to: ", quicServerAddr)
	listener, err := quic.ListenAddr(quicServerAddr, generateTLSConfig(), quicConfig)
	HandleError(err)

	fmt.Println("Server started! Waiting for streams from client...")

	sess, err := listener.Accept() //accepting connection from sender
	HandleError(err)

	fmt.Println("session created: ", sess.RemoteAddr())

	streamReceive, err := sess.AcceptStream()
	HandleError(err)

	defer streamReceive.Close()

	fmt.Println("streamReceive created: ", streamReceive.StreamID())
	//data1 := make([]byte, 2048)
	//length1,err := streamReceive.Read(data1)

	//fileName := string(data1[:length1])
	//fmt.Println(fileName)
	f, err := os.Create("10MB.bin")

	if err != nil {
		fmt.Println(err)
		return
	}
	HandleError(err)

	defer f.Close()

	totallength := 0
	ticker := time.NewTimer(time.Second * 1)
	for {
		data := make([]byte, 2048) // size is needed to make use of ReadFull(). ReadAll() needs EOF to stop accepting while ReadFull just needs the fixed size.

		length, err := streamReceive.Read(data)
		totallength += length

		select {
		case <-ticker.C:
			fmt.Println(float64(totallength) / float64(1024*128))
			ticker.Reset(time.Second * 1)
			totallength = 0
		default:

		}
		HandleError(err)

		if string(data[:length]) == "finish" {
			fmt.Println("传输完成")
			break
		}
		f.Write(data[:length])

	}

}

func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}
