package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

var (
	username = "ddd"
	password = "06092002"
	hostname = "127.0.0.1"
	port     = "3000"
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	conn *websocket.Conn

	send chan []byte
}

func (c *Client) readPump(sess *ssh.Session, stdin *io.WriteCloser) {
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
		old := os.Stdout
		oldE := os.Stderr
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w
		sess.Stdout = os.Stdout
		sess.Stderr = os.Stderr
		_, err = fmt.Fprintf(*stdin, "%s\n", message)
		if err != nil {
			log.Printf("error: %v", err)
		}
		outC := make(chan string)
		go func() {
			var buf bytes.Buffer
			io.Copy(&buf, r)
			outC <- buf.String()
		}()
		w.Close()
		os.Stdout = old
		os.Stderr = oldE
		out := <-outC
		c.send <- []byte(out)
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
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshClient, err := ssh.Dial("tcp", hostname+":"+port, config)
	if err != nil {
		log.Fatal(err)
	}
	defer sshClient.Close()
	sess, err := sshClient.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}
	defer sess.Close()
	stdin, err := sess.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	err = sess.Shell()
	if err != nil {
		log.Printf("error: %v", err)
	}
	go client.writePump()
	go client.readPump(sess, &stdin)
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
