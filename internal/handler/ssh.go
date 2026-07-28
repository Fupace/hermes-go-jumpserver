package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/Fupace/hermes-go-jumpserver/internal/store"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type SSHHandler struct {
	store *store.Store
}

func NewSSHHandler(s *store.Store) *SSHHandler {
	return &SSHHandler{store: s}
}

func (h *SSHHandler) HandleSSH(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/ssh/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing machine ID", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "invalid machine ID", http.StatusBadRequest)
		return
	}

	machine, err := h.store.GetMachine(id)
	if err != nil || machine == nil {
		http.Error(w, "machine not found", http.StatusNotFound)
		return
	}

	cols := 80
	rows := 24
	if c := r.URL.Query().Get("cols"); c != "" {
		cols, _ = strconv.Atoi(c)
	}
	if rw := r.URL.Query().Get("rows"); rw != "" {
		rows, _ = strconv.Atoi(rw)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	config := &ssh.ClientConfig{
		User:            machine.Username,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if machine.Password != "" {
		config.Auth = append(config.Auth, ssh.Password(machine.Password))
	}
	if machine.KeyFile != "" {
		keyData, err := base64.StdEncoding.DecodeString(machine.KeyFile)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(keyData)
			if err == nil {
				config.Auth = append(config.Auth, ssh.PublicKeys(signer))
			}
		}
	}

	addr := fmt.Sprintf("%s:%d", machine.Host, machine.Port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nSSH connection failed: %v\r\n", err)))
		return
	}
	defer sshClient.Close()

	session, err := sshClient.NewSession()
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to create session: %v\r\n", err)))
		return
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to request PTY: %v\r\n", err)))
		return
	}

	stdinPipe, _ := session.StdinPipe()
	stdoutPipe, _ := session.StdoutPipe()
	stderrPipe, _ := session.StderrPipe()

	if err := session.Shell(); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to start shell: %v\r\n", err)))
		return
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		defer stdinPipe.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var wsMsg struct {
				Type string `json:"type"`
				Data string `json:"data"`
			}
			if err := json.Unmarshal(msg, &wsMsg); err == nil {
				switch wsMsg.Type {
				case "stdin":
					stdinPipe.Write([]byte(wsMsg.Data))
				case "resize":
					var resize struct {
						Cols int `json:"cols"`
						Rows int `json:"rows"`
					}
					json.Unmarshal([]byte(wsMsg.Data), &resize)
					if resize.Cols > 0 && resize.Rows > 0 {
						session.WindowChange(resize.Rows, resize.Cols)
					}
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				conn.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		io.Copy(&wsWriter{conn: conn}, stderrPipe)
	}()

	wg.Wait()
}

type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}
