package main

import (
	"bytes"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"image"
	"image/png"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"
)

func main() {
	if err := resumeLogin(); err != nil {
		log.Println(err, "准备登录")
		newLogin()
	}
	log.Println("登录成功")
	select {}
}

var bClient *client.QQClient

const (
	dataToken  = "session.token"
	dataDevice = "device.json"
)

func newLogin() error {
	client.GenRandomDevice()
	rsp, err := bClient.FetchQRCodeCustomSize(1, 2, 1)
	if err != nil {
		log.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(rsp.ImageData))
	if err != nil {
		log.Fatal(err)
	}
	data := img.(*image.Gray).Pix
	bound := img.Bounds().Max.X
	buf := make([]byte, 0, (bound*3+1)*bound)
	for y := 0; y < bound; y++ {
		i := y * bound
		for x := 0; x < bound; x++ {
			if data[i] == 255 {
				buf = append(buf, "█ "...)
			} else {
				buf = append(buf, "  "...)
			}
			i++
		}
		buf = append(buf, '\n')
	}
	os.Stdout.Write(buf)
	s, err := bClient.QueryQRCodeStatus(rsp.Sig)
	if err != nil {
		return err
	}
	prevState := s.State
	for {
		time.Sleep(time.Second)
		s, _ = bClient.QueryQRCodeStatus(rsp.Sig)
		if s == nil {
			continue
		}
		if prevState == s.State {
			continue
		}
		prevState = s.State
		switch s.State {
		case client.QRCodeCanceled:
			log.Fatalf("扫码被用户取消.")
		case client.QRCodeTimeout:
			log.Fatalf("二维码过期")
		case client.QRCodeWaitingForConfirm:
			log.Println("扫码成功, 请在手机端确认登录.")
		case client.QRCodeConfirmed:
			_, err := bClient.QRCodeLogin(s.LoginInfo)
			os.WriteFile(dataDevice, client.SystemDeviceInfo.ToJson(), 0644)
			os.WriteFile(dataToken, bClient.GenToken(), 0644)
			return err
		case client.QRCodeImageFetch, client.QRCodeWaitingForScan:
			// ignore
		}
	}
}

func resumeLogin() error {
	bClient = client.NewClientEmpty()
	bClient.GroupMessageEvent.Subscribe(GroupHandler)
	bClient.DisconnectedEvent.Subscribe(DisconnectedEvent)
	dev, err := os.ReadFile(dataDevice)
	if err != nil {
		return err
	}
	err = client.SystemDeviceInfo.ReadJson(dev)
	if err != nil {
		return err
	}
	token, err := os.ReadFile(dataToken)
	if err != nil {
		return err
	}
	return bClient.TokenLogin(token)
}

var hifiSBReply []string

const hifiSB = "hifiSB"

func DisconnectedEvent(c *client.QQClient, e *client.ClientDisconnectedEvent) {
	log.Println("Disconnected", e.Message)
}

func refreshImg() error {
	dir, err := os.Open(hifiSB)
	if err != nil {
		return err
	}
	defer dir.Close()
	list, err := dir.ReadDir(0)
	if err != nil {
		return err
	}
	hifiSBReply = make([]string, len(list))
	for i, f := range list {
		hifiSBReply[i] = f.Name()
	}
	return nil
}

func GroupHandler(c *client.QQClient, e *message.GroupMessage) {
	if runtime.GOOS != "linux" {
		log.Println(e.GroupCode, e.Sender.Uin, e.ToString())
	}
	//if e.Sender.Uin == 767763591 {
	//	if len(e.Elements) > 0 {
	//		if elem, ok := e.Elements[0].(*message.TextElement); ok {
	//			if elem.Content == "hifiSB" {
	//				err := refreshImg()
	//				if err != nil {
	//					c.SendGroupMessage(e.GroupCode, message.NewSendingMessage().Append(message.NewText(err.Error())))
	//				}
	//			}
	//			if elem.Content == "在吗" {
	//				c.SendGroupMessage(e.GroupCode, message.NewSendingMessage().Append(message.NewText("buzai")))
	//			}
	//		}
	//	}
	//}
	if e.GroupCode == 558524420 && (e.Sender.Uin == 3406758965 || e.Sender.Uin == 2388843095) {
		log.Println("狗叫", e.ToString())
		if rand.Int31()%4 == 0 {
			return
		}
		if len(hifiSBReply) == 0 {
			err := refreshImg()
			if err != nil {
				c.SendGroupMessage(e.GroupCode, message.NewSendingMessage().Append(message.NewText(err.Error())))
			}
			if len(hifiSBReply) == 0 {
				return
			}
		}
		i := rand.Intn(len(hifiSBReply))
		f, err := os.Open(hifiSB + "/" + hifiSBReply[i])
		if err != nil {
			c.SendGroupMessage(e.GroupCode, message.NewSendingMessage().Append(message.NewText(err.Error())))
			return
		}
		img, err := c.UploadGroupImage(e.GroupCode, f)
		f.Close()
		if err == nil {
			c.SendGroupMessage(e.GroupCode, message.NewSendingMessage().Append(img))
		}
	}
}
