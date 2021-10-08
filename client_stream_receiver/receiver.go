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
	"io"
	"log"
	"math/big"
	random "math/rand"
	"os"
	"time"
)

//The reciever function that recieves the frames from the sender
//input args - the directory to store the frames. Run the viewer function to show the video

const quicServerAddr = "127.0.0.1:5252"

var elapsed time.Duration
var size int64

func HandleError(err error) {
	if err != nil {
		fmt.Println("App elapsed: ", elapsed)
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

func main() {

	window := gocv.NewWindow("Capture Window")
	defer window.Close()
	quicConfig := &quic.Config{
		CreatePaths: true,
	}
	f, err := os.OpenFile("client_stream_receiver/clientlog.txt", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Println() //4G + 4G or 4G or 4G + wifi or wifi?

	// initializing mpquic server
	sess, err := quic.DialAddr(quicServerAddr, &tls.Config{InsecureSkipVerify: true}, quicConfig)
	HandleError(err)

	stream, err := sess.AcceptStream()
	HandleError(err)
	defer stream.Close()

	fmt.Println("stream created: ", stream.StreamID())

	frame_counter := 0
	t := time.Now()
	var t1, t2 time.Duration = 0, 0
	for {
		if frame_counter%20 == 0 {
			t = time.Now()
			size = 0
		}
		siz := make([]byte, 8) // size is needed to make use of ReadFull(). ReadAll() needs EOF to stop accepting while ReadFull just needs the fixed size.

		_, err := io.ReadFull(stream, siz)      //recieve the size
		data := binary.LittleEndian.Uint64(siz) //if the first few bytes contain the length; else use BigEndian or reverse the byte[] and use LittleEndian
		HandleError(err)

		if data == 0 {
			defer stream.Close()
			return
		}
		t4 := time.Now()
		buff := make([]byte, data)
		size += int64(data)
		len2, err := io.ReadFull(stream, buff) // recieve image

		t2 += time.Since(t4)
		HandleError(err)

		//if empty buffer
		if len2 == 0 {
			defer stream.Close()
			return
		}

		//calculate the time of this image from the webcam to this client
		imgbuff := buff[0 : len2-8]
		timeStamp := buff[len2-8 : len2]
		clientTime := binary.LittleEndian.Uint64(timeStamp)
		//fmt.Println(clientTime)
		//fmt.Println(" the time :",uint64(time.Now().UnixMilli())-clientTime)//the time consumed

		img, err := gocv.IMDecode(imgbuff, 1) //IMReadFlag 1 ensure that image is converted to 3 channel RGB

		HandleError(err)
		// if decoding fails

		if img.Empty() {
			defer stream.Close()
			return
		}

		random.Seed(time.Now().UnixNano())
		num := random.Intn(10)
		if num == 5 {
			clientTime++
			//log.Println("time consumed :", uint64(time.Now().UnixMilli())-clientTime)
		}

		t3 := time.Now()

		window.IMShow(img)

		t1 += time.Since(t3)

		if window.WaitKey(1) == 27 {
			break
		}

		frame_counter += 1
		if frame_counter%20 == 0 {
			elapsed = time.Since(t)
			duration := float64(elapsed) / float64(time.Second)
			log.Println("FPS:", 20/(duration))
			log.Println("throughput(MB):", float64(size)/(1024.0*1024.0*float64(duration)))
			log.Println("gocv time :", t1, "transfer time :", t2, "total time:", elapsed)
			t1, t2 = 0, 0
			//log.Println("FPS:",100 / (int(elapsed / time.Second)))
		}
		/*fmt.Println(videoDir + "/img" + strconv.Itoa(frame_counter) + ".jpg")

		file.Close()*/
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
