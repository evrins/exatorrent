package core

import (
	"encoding/json"
	"fmt"
	"github.com/anacrolix/torrent"
	"log/slog"
	"net/http"
)

type AddMagnetRequest struct {
	Uri       string `json:"uri"`
	AutoStart bool   `json:"autoStart"`
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
	AddFromSpec("adminuser", spec, !req.AutoStart, false)

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
		Msg:   fmt.Sprintf("delete torrent %s successfully", req.Hash),
	}
	handleResponse(w, r, resp)
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var resp = Resp{Type: "resp", State: "error", Msg: err.Error()}
	handleResponse(w, r, resp)
}

func handleResponse(w http.ResponseWriter, r *http.Request, resp Resp) {
	var err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Info("fail to send response. err: %v", err)
	}
}
