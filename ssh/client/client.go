package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	conn *websocket.Conn

	send chan []byte
}

func (c *Client) readPump() {
	username := ""
	password := ""
	hostname := "127.0.0.1"
	port := "3000"
	auth := 0
	defer func() {
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		if auth == 0 {
			username = string(message)
			auth++
			continue
		}
		if auth == 1 {
			password = string(message)
			auth++
		}
		config := &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		sshClient, err := ssh.Dial("tcp", hostname+":"+port, config)
		if err != nil {
			log.Printf("error3: %v", err)
			auth = 0
			c.send <- []byte("wrong_reg")
			continue
		}
		defer sshClient.Close()
		if auth == 2 {
			auth++
			sshClient.Close()
			continue
		}
		sess, err := sshClient.NewSession()
		if err != nil {
			log.Printf("error4: %v", err)
		}
		defer sess.Close()
		var b bytes.Buffer
		sess.Stdout = &b
		err = sess.Run(string(message))
		if err != nil {
			log.Printf("error1: %v", err)
		}
		if err != nil {
			log.Printf("error2: %v", err)
		}
		bb := b.Bytes()
		sshClient.Close()
		sess.Close()
		c.send <- bb
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256)}
	go client.writePump()
	go client.readPump()
}

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(w, r)
	})
	err := http.ListenAndServe("127.0.0.1:6060", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
