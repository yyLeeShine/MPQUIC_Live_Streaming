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
	size     int64
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
	deviceID := 0                          //os.Args[1]// device id for the webcam, 0 be default
	quicServerAddr := "1.116.187.145:4242" //os.Args[2]// the server address, in this case 0.0.0.0:4242

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
	var t1, t2 time.Duration = 0, 0

	for {

		if image_count%100 == 0 { //fps is calculated if 100 image is transmitted
			t = time.Now()
			size = 0
		}
		if image_count == 1000 {
			break
		}

		// read the image from the device
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		t3 := time.Now()                     //t3 is to calculate the time  gocv consumed when an image is endoded
		buf, _ := gocv.IMEncode(".jpg", img) // encode the imgae into byte[] for transport
		buf2 := buf.GetBytes()

		timeStamp := make([]byte, 8) //the current time
		binary.LittleEndian.PutUint64(timeStamp, uint64(time.Now().UnixMilli()))
		buf2 = append(buf2, timeStamp...)
		length = len(buf2)
		size += int64(length)
		t1 += time.Since(t3)

		bs := make([]byte, 8)
		binary.LittleEndian.PutUint32(bs, uint32(length)) //encoding the length(integer) as a byte[] for transport

		fmt.Println(image_count)

		image_count = image_count + 1

		stream.Write(bs) //sends the length of the frame so that appropriate buffer size can be created in the reciever side

		time.Sleep(time.Second / 1000) //time delay of 10 milli second

		t4 := time.Now()
		stream.Write(buf2) //sends the frame
		t2 += time.Since(t4)
		if image_count%100 == 0 {
			elapsed := time.Since(t)
			duration := int(elapsed / time.Second)
			log.Println("FPS:", 100/(duration))
			log.Println("throughput(MB):", float64(size)/(1024.0*1024.0*float64(duration)))
			log.Println("gocv time :", t1, "transfer time :", t2, "total time:", elapsed)
			t1, t2 = 0, 0
		}
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
