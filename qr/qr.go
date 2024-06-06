package qr

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"wzj_sign/db"
	"wzj_sign/model"
)

var done chan interface{}
var interrupt chan os.Signal

var wsUrl string = "https://www.teachermate.com.cn/faye"
var clientID string = ""

func InitQrSign(courseId int, signId int){
	post_data := `[{"channel":"/meta/handshake","version":"1.0","supportedConnectionTypes":["websocket","eventsource","long-polling","cross-origin-long-polling","callback-polling"],"id":"1"}]`
	data := strings.NewReader(post_data)
	req, err := http.NewRequest("POST", wsUrl, data)
	if err != nil {
		log.Println("Error creating initClientID request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Set("Host", "v18.teachermate.cn")
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
        log.Println("Error sending initClientID request:", err)
        return
    }
    defer response.Body.Close()
	
	body, _ := io.ReadAll(response.Body)
	log.Println("initClientID Response:", string(body))
	var wsData []model.WSData
	json.Unmarshal(body, &wsData)
	clientID = wsData[0].ClientID

	SubscribeToQRSign(courseId, signId)
}

func SubscribeToQRSign(courseId int, signId int){
	if clientID == ""{
		log.Fatalln("Not init clientID yet")
	}
	post_data := fmt.Sprintf(`[{"channel":"/meta/connect","clientId":"%s","connectionType":"long-polling","id":"2","advice":{"timeout":0}},
							{"channel":"/meta/subscribe","clientId":"%s","subscription":"/attendance/%d/%d/qr","id":"3"}]`, 
							clientID, clientID, courseId, signId)
	data := strings.NewReader(post_data)
	req, err := http.NewRequest("POST", wsUrl, data)
	if err != nil {
		log.Println("Error creating initClientID request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Set("Host", "v18.teachermate.cn")
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
        log.Println("Error sending SubscribeToQRSign request:", err)
        return
    }
    defer response.Body.Close()
	
	body, _ := io.ReadAll(response.Body)
	log.Println("SubscribeToQRSign Response:", string(body))
	var wsData []model.WSData
	json.Unmarshal(body, &wsData)
	if wsData[1].Successful {
		log.Println("OK")
		Start(signId)
	}
}

func receiveHandler(connection *websocket.Conn, signId int) {
    defer close(done)
    for {
        _, msg, err := connection.ReadMessage()
        if err != nil {
            log.Println("Error in receive:", err)
            return
        }
		if strings.Contains(string(msg), "qrUrl") {
			var qrData []model.QRCodeUrlData
			json.Unmarshal(msg, &qrData)
			qrCodeUrl := qrData[0].Data.QrURL
			// log.Println("getting qrURL", qrCodeUrl)
			result := db.RedisSet("wzj:qr:" + fmt.Sprint(signId), qrCodeUrl, 30 * time.Second)
			if result.Err() != nil {
				log.Println("Error setting key:", result.Err())
				return
			}
		}
    }
}

func Start(signId int) {
    done = make(chan interface{}) // Channel to indicate that the receiverHandler is done
    interrupt = make(chan os.Signal) // Channel to listen for interrupt signal to terminate gracefully

    signal.Notify(interrupt, os.Interrupt) // Notify the interrupt channel for SIGINT

    socketUrl := "wss://www.teachermate.com.cn/faye"
    conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
    if err != nil {
        log.Fatal("Error connecting to Websocket Server:", err)
    }
    defer conn.Close()
    go receiveHandler(conn, signId)

	var counter = 3

    // Our main loop for the client
    // We send our relevant packets here
    for {
        select {
			case <-time.After(time.Duration(5) * time.Millisecond * 1000):
            // Send an echo packet every second
			counter = counter + 1
			var connectString = fmt.Sprintf(`[{"channel":"/meta/connect","clientId":"%s","connectionType":"websocket","id":"%d"}]`, clientID, counter)
			err := conn.WriteMessage(websocket.TextMessage, []byte(connectString))
            if err != nil {
                log.Println("Error during writing to websocket:", err)
                return
            }

        case <-interrupt:
            // We received a SIGINT (Ctrl + C). Terminate gracefully...
            log.Println("Received SIGINT interrupt signal. Closing all pending connections")

            // Close our websocket connection
            err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
            if err != nil {
                log.Println("Error during closing websocket:", err)
                return
            }

            select {
            case <-done:
                log.Println("Receiver Channel Closed! Exiting....")
            case <-time.After(time.Duration(1) * time.Second):
                log.Println("Timeout in closing receiving channel. Exiting....")
            }
            return
        }
	}
}