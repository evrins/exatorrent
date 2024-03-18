package core

import "net/http"

func ListUsers(w http.ResponseWriter, r *http.Request) {
	handleResponseBytes(w, r, MainHub.ListUsers())
}
