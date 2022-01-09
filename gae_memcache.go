package main

import (
	"encoding/json"
	"google.golang.org/appengine"
	"google.golang.org/appengine/memcache"
	"log"
	"net/http"
	"time"
)

type BaseRequest struct {
	Key string
}

type SetRequest struct {
	BaseRequest
	Value string
}

type ExpireRequest struct {
	BaseRequest
	Expiration int
}

func gaeLog(msg string, a ...interface{}) {
	log.Printf("[MEMCACHE] " + msg, a);
}

func decode(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	return true
}

func main() {
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/set", setHandler)
	http.HandleFunc("/expire", expireHandler)

	appengine.Main()
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	var req BaseRequest
	if !decode(w, r, &req) {
		return
	}

	result, err := memcache.Get(r.Context(), req.Key)
	switch err {
	case memcache.ErrCacheMiss: {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	case nil: {
		if result == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err = w.Write(result.Value); err != nil {
			gaeLog("Failed to write to socket: %s", err)
		}
		gaeLog("Got %s in memcache: %s", req.Key, result.Value)
		return
	}
	default: {
		w.WriteHeader(http.StatusInternalServerError)
		gaeLog("Failed to get value from memcache: %s", err)
		return
	}
	}
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	var req SetRequest
	if !decode(w, r, &req) {
		return
	}

	gaeLog("Setting %s to %s", req.Key, req.Value)

	item := &memcache.Item{
		Key: req.Key,
		Value: []byte(req.Value),
	}

	if err := memcache.Set(r.Context(), item); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		gaeLog("Failed to set value in memcache: %s -> %s", item.Key, item.Value)
		return
	}

	gaeLog("Set %s in memcache to %s", item.Key, item.Value)
}

func expireHandler(w http.ResponseWriter, r *http.Request) {
	var req ExpireRequest
	if !decode(w, r, &req) {
		return
	}

	gaeLog("Changing expiry of %s", req.Key)

	cur, err := memcache.Get(r.Context(), req.Key)
	switch err {
	case memcache.ErrCacheMiss: {
		w.WriteHeader(http.StatusNotFound)
		gaeLog("Attempted to change expiry on non-existent item %s", req.Key)
		return
	}
	case nil: {
		break
	}
	default: {
		w.WriteHeader(http.StatusInternalServerError)
		gaeLog("Error setting expiration: %s", err)
		return
	}
	}

	exp  := time.Duration(req.Expiration) * time.Second
	item := &memcache.Item{
		Key: cur.Key,
		Value: cur.Value,
		Expiration: exp,
	}

	if err2 := memcache.Set(r.Context(), item); err2 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		gaeLog("Failed to set value with expiration in memcache: %s -> %s", item.Key, item.Expiration)
		return
	}

	gaeLog("Set expiration of %s to %s seconds from now", item.Key, item.Expiration)
	w.WriteHeader(http.StatusOK)
}