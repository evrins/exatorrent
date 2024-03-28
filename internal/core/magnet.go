package core

import (
	"encoding/json"
	"fmt"
	"github.com/anacrolix/torrent"
	"log/slog"
	"net/http"
	"path/filepath"
)

type AddMagnetRequest struct {
	Uri       string `json:"uri"`
	AutoStart bool   `json:"autoStart"`
}

func GetMagnet(w http.ResponseWriter, r *http.Request) {
	var buf = GetTorrents(Engine.TUDb.ListTorrents("adminuser"))
	handleResponseBytes(w, r, buf)
}

func AddMagnet(w http.ResponseWriter, r *http.Request) {
	var req AddMagnetRequest
	var err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		handleError(w, r, err)
		return
	}
	spec, err := torrent.TorrentSpecFromMagnetUri(req.Uri)
	if err != nil {
		handleError(w, r, err)
		return
	}

	// hard coded here will remove user system
	go AddFromSpec("adminuser", spec, !req.AutoStart, false)

	var resp = Resp{
		Type:     "resp",
		State:    "success",
		Infohash: spec.InfoHash.HexString(),
		Msg:      "Torrent Spec Added",
	}
	handleResponse(w, r, resp)
}

type RemoveMagnetRequest struct {
	Hash string `json:"hash"`
}

func RemoveMagnet(w http.ResponseWriter, r *http.Request) {
	var req RemoveMagnetRequest
	var err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		handleError(w, r, err)
		return
	}
	ih, err := MetaFromHex(req.Hash)
	if err != nil {
		handleError(w, r, err)
		return
	}
	RemoveTorrent("", ih)
	var resp = Resp{
		Type:  "resp",
		State: "success",
		Msg:   fmt.Sprintf("remove torrent %s successfully", req.Hash),
	}
	handleResponse(w, r, resp)
}

type DeleteMagnetRequest struct {
	Hash string `json:"hash"`
}

func DeleteMagnet(w http.ResponseWriter, r *http.Request) {
	var req DeleteMagnetRequest
	var err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		handleError(w, r, err)
	}

	ih, err := MetaFromHex(req.Hash)
	if err != nil {
		handleError(w, r, err)
		return
	}
	DeleteTorrent("adminuser", ih)

	var resp = Resp{
		Type:  "resp",
		State: "success",
		Msg:   fmt.Sprintf("delete torrent %s successfully", req.Hash),
	}
	handleResponse(w, r, resp)
}

func GetFsDirInfoApi(w http.ResponseWriter, r *http.Request) {
	var hash = r.URL.Query().Get("hash")
	var dir = r.URL.Query().Get("dir")

	ih, err := MetaFromHex(hash)
	if err != nil {
		handleError(w, r, err)
	}
	var buf = GetFsDirInfo(ih, filepath.Clean(dir))
	handleResponseBytes(w, r, buf)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var resp = Resp{Type: "resp", State: "error", Msg: err.Error()}
	w.Header().Set("Content-Type", "application/json")
	handleResponse(w, r, resp)
}

func handleResponse(w http.ResponseWriter, r *http.Request, resp Resp) {
	w.Header().Set("Content-Type", "application/json")
	var err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("fail to send response. err: %v", err)
	}
}

func handleResponseBytes(w http.ResponseWriter, r *http.Request, buf []byte) {
	w.Header().Set("Content-Type", "application/json")
	var _, err = w.Write(buf)
	if err != nil {
		slog.Error("fail to send response. err: %v", err)
	}
}
