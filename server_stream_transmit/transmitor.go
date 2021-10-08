package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	quic "github.com/lucas-clemente/quic-go"
	"io"
	"math/big"
	"os"
)

//The reciever function that recieves the frames from the sender
//input args - the directory to store the frames. Run the viewer function to show the video

const quicServerAddr = "0.0.0.0:4242"
const quicServerAddr2 = "0.0.0.0:5252"

func HandleError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

func main() {

	quicConfig := &quic.Config{
		CreatePaths: true,
	}
	fmt.Println("Attaching to: ", quicServerAddr2)
	listener2, err := quic.ListenAddr(quicServerAddr2, generateTLSConfig(), quicConfig) //因为服务器才需要公钥和私钥
	HandleError(err)

	sess2, err := listener2.Accept() //accepting a session from sender
	HandleError(err)

	fmt.Println("session created: ", sess2.RemoteAddr())

	streamSender, err := sess2.OpenStream()
	HandleError(err)

	defer streamSender.Close()

	fmt.Println("streamReceive created: ", streamSender.StreamID())
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

	//var sliceChan chan []byte = make(chan []byte)
	//var sliceChan2 chan []byte = make(chan []byte)
	//go send(streamSender,sliceChan,sliceChan2)
	for {
		siz := make([]byte, 8) // size is needed to make use of ReadFull(). ReadAll() needs EOF to stop accepting while ReadFull just needs the fixed size.

		_, err := io.ReadFull(streamReceive, siz) //recieve the size

		data := binary.LittleEndian.Uint64(siz) //if the first few bytes contain the length; else use BigEndian or reverse the byte[] and use LittleEndian
		HandleError(err)
		//streamSender.Write(siz)
		//sliceChan<-siz
		if data == 0 {
			defer streamReceive.Close()
			return
		}

		buff := make([]byte, data)
		len2, err := io.ReadFull(streamReceive, buff) // recieve image
		//sliceChan2<-buff
		HandleError(err)

		//if empty buffer
		if len2 == 0 {
			defer streamReceive.Close()
			return
		}
		streamSender.Write(siz)
		//		time.Sleep(time.Second/100)
		//streamSender.Write(buff)

	}

}
func send(stream quic.Stream, sliceChan1 chan []byte, sliceChan2 chan []byte) {
	for {
		siz := <-sliceChan1
		(stream).Write(siz)
		data := <-sliceChan2
		(stream).Write(data)

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
