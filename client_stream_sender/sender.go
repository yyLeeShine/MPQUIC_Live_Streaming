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
	"gocv.io/x/gocv"
	"log"
	"math/big"
	"os"
	"time"
)

func HandleError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

var (
	deviceID int
	err      error
	webcam   *gocv.VideoCapture
)

//a sender function that generates frames and sends them over mpquic to the reciever.
//input args - deviceID and mpquic-server address

func main() {

	f, err := os.OpenFile("clientlog.txt", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	log.SetOutput(f)
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Println()
	//if len(os.Args) < 3 {
	//	fmt.Println("How to run:\n\tmjpeg-streamer [camera ID] [host:port]")
	//	return
	//}

	// parse args
	deviceID := 0                      //os.Args[1]// device id for the webcam, 0 be default
	quicServerAddr := "127.0.0.1:4242" //os.Args[2]// the server address, in this case 0.0.0.0:4242

	//open webcam
	webcam, err = gocv.OpenVideoCapture(deviceID)

	if err != nil {
		fmt.Printf("Error opening capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	//mpquic server
	quicConfig := &quic.Config{
		CreatePaths: true,
	}

	sess, err := quic.DialAddr(quicServerAddr, &tls.Config{InsecureSkipVerify: true}, quicConfig)
	HandleError(err)

	stream, err := sess.OpenStream()
	HandleError(err)

	defer stream.Close()

	var length = 0

	//an infinite loop that generates frames from the webcam and sends to reciever

	img := gocv.NewMat()
	defer img.Close()

	var image_count = 0
	t := time.Now()

	for {
		if image_count%100 == 0 {
			t = time.Now()
		}
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		buf, _ := gocv.IMEncode(".jpg", img) // encode the imgae into byte[] for transport
		buf2 := buf.GetBytes()
		length = len(buf2)

		bs := make([]byte, 60)
		binary.LittleEndian.PutUint32(bs, uint32(length)) //encoding the length(integer) as a byte[] for transport

		fmt.Println(image_count)

		image_count = image_count + 1

		stream.Write(bs) //sends the length of the frame so that appropriate buffer size can be created in the reciever side

		time.Sleep(time.Second / 100) //time delay of 10 milli second
		if image_count%100 == 0 {
			elapsed := time.Since(t)
			log.Println(100 / (int(elapsed / time.Second)))
		}

		stream.Write(buf2) //sends the frame
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
