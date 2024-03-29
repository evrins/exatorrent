package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/anacrolix/torrent/metainfo"

	"github.com/gorilla/websocket"
)

// Hub
type Hub struct {
	sync.RWMutex
	Conns map[string]map[string]*UserConn // connection per user per session
}

func (h *Hub) Add(uc *UserConn) {
	var username = uc.Username
	var token = uc.Token
	conns, ok := h.Conns[username]
	if ok {
		conn, ok := conns[token]
		if ok {
			conn.Close()
		}

		h.Lock()
		h.Conns[username][token] = uc
		h.Unlock()
	} else {
		h.Lock()
		h.Conns[username] = map[string]*UserConn{
			uc.Token: uc,
		}
		h.Unlock()
	}

	Info.Printf("User %s (%s) Connected\n", username, uc.Conn.RemoteAddr().String())
}

func (h *Hub) Get(username string) map[string]*UserConn {
	conns, ok := h.Conns[username]
	if !ok {
		return nil
	}
	return conns
}

func (h *Hub) StopStreamByUser(username string) {
	conns, ok := h.Conns[username]
	if !ok {
		return
	}

	for _, conn := range conns {
		conn.StopStream()
	}
}

func (h *Hub) SendMsg(User string, Type string, State string, Resp string) {
	if User == "" {
		return
	}

	conns, ok := h.Conns[User]
	if !ok {
		return
	}

	for _, conn := range conns {
		_ = conn.SendMsg(Type, State, Resp)
	}
}

func (h *Hub) SendMsgU(User string, Type string, hash string, State string, resp string) {
	var et = Empty
	switch resp {
	case "Torrent Spec Added":
		et = Added
	case "Torrent is Loaded":
		et = Loaded
	case "Torrent Started":
		et = Started
	case "Torrent Completed":
		et = Completed
	case "Torrent Stopped":
		et = Stopped
	case "Torrent Removed":
		et = Removed
	case "Torrent Deleted":
		et = Deleted
	default:
		et = Empty
	}

	if et != Empty {
		go PublishEvent(hash, et)
	}

	if User == "" {
		return
	}

	conns, ok := h.Conns[User]
	if !ok {
		return
	}

	for _, conn := range conns {
		_ = conn.SendMsgU(Type, State, hash, resp)
	}
}

func (h *Hub) Remove(uc *UserConn) {
	if uc == nil {
		return
	}
	h.Lock()
	defer h.Unlock()
	conns, ok := h.Conns[uc.Username]
	if !ok {
		return
	}
	delete(conns, uc.Token)
	Info.Printf("User %s (%s) Disconnected\n", uc.Username, uc.Conn.RemoteAddr().String())
}

func (h *Hub) RemoveUser(Username string) {
	conns, ok := h.Conns[Username]
	if !ok {
		return
	}
	for _, conn := range conns {
		conn.Close()
	}
}

func (h *Hub) ListUsers() (ret []byte) {
	var userconnmsg []*UserConnMsg
	h.Lock()
	defer h.Unlock()
	for name, conns := range h.Conns {
		if conns == nil {
			continue
		}
		for _, conn := range conns {
			var usermsg UserConnMsg

			usermsg.Username = name
			usermsg.Token = conn.Token
			usermsg.IsAdmin = conn.IsAdmin
			usermsg.Time = conn.Time
			usermsg.RemoteAddr = conn.Conn.RemoteAddr().String()

			userconnmsg = append(userconnmsg, &usermsg)
		}
	}
	ret, _ = json.Marshal(DataMsg{Type: "userconn", Data: userconnmsg})
	return
}

var MainHub Hub = Hub{
	RWMutex: sync.RWMutex{},
	Conns:   make(map[string]map[string]*UserConn),
}

// UserConn
type UserConn struct {
	Sendmu    sync.Mutex
	Username  string
	Token     string
	IsAdmin   bool
	Time      time.Time
	Conn      *websocket.Conn
	Stream    sync.Mutex
	Streamers MutInt
}

func NewUserConn(Username, token string, Conn *websocket.Conn, IsAdmin bool) (uc *UserConn) {
	uc = &UserConn{
		Username: Username,
		Token:    token,
		Conn:     Conn,
		IsAdmin:  IsAdmin,
		Time:     time.Now(),
	}
	MainHub.Add(uc)
	return
}

func (uc *UserConn) SendMsg(Type string, State string, Msg string) (err error) {
	resp, _ := json.Marshal(Resp{Type: Type, State: State, Msg: Msg})
	err = uc.Send(resp)
	return
}

func (uc *UserConn) SendMsgU(Type string, State string, hash string, msg string) (err error) {
	resp, _ := json.Marshal(Resp{Type: Type, State: State, Infohash: hash, Msg: msg})
	err = uc.Send(resp)
	return
}

func (uc *UserConn) Send(v []byte) (err error) {
	uc.Sendmu.Lock()
	_ = uc.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	err = uc.Conn.WriteMessage(websocket.TextMessage, v)
	uc.Sendmu.Unlock()
	if err != nil {
		Err.Println(err)
		uc.Close()
		return
	}
	return
}

func (uc *UserConn) StopStream() {
	uc.Streamers.Inc()
	uc.Stream.Lock()
	Info.Println("Stopped Stream for ", uc.Username)
	uc.Stream.Unlock()
	uc.Streamers.Dec()
}

func (uc *UserConn) Close() {
	uc.Sendmu.Lock()
	_ = uc.Conn.Close()
	uc.Sendmu.Unlock()
	MainHub.Remove(uc)
}

var hc = http.Client{
	Timeout: 10 * time.Second,
}

func sendPostReq(h metainfo.Hash, url string, name string) {
	Info.Println("Torrent ", h, " has completed. Sending POST request to ", url)
	postrequest := struct {
		Metainfo metainfo.Hash `json:"metainfo"`
		Name     string        `json:"name"`
		State    string        `json:"state"`
		Time     time.Time     `json:"time"`
	}{
		Metainfo: h,
		Name:     name,
		Time:     time.Now(),
		State:    "torrent-completed-exatorrent",
	}

	jsonData, err := json.Marshal(postrequest)

	if err != nil {
		Warn.Println(err)
		return
	}

	resp, err := hc.Post(url, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		Warn.Println("POST Request failed to Send. Hook failed")
		Warn.Println(err)
		return
	}

	if resp != nil {
		resp.Body.Close()
	}

	Info.Println("POST Request Sent. Hook Succeeded")
}
