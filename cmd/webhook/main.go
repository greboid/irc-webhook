package main

import (
	"encoding/json"
	"errors"
	"flag"
	"github.com/greboid/irc/v2/logger"
	"github.com/greboid/irc/v2/plugins"
	"github.com/greboid/irc/v2/rpc"
	"github.com/kouhin/envflag"
	"net/http"
	"strings"
)

var (
	RPCHost  = flag.String("rpc-host", "localhost", "gRPC server to connect to")
	RPCPort  = flag.Int("rpc-port", 8001, "gRPC server port")
	RPCToken = flag.String("rpc-token", "", "gRPC authentication token")
	Channel  = flag.String("channel", "", "Channel to send messages to")
	Debug    = flag.Bool("debug", false, "Show debugging info")

	DBPath        = flag.String("db-path", "/data/db", "Path to token database")
	AdminKey      = flag.String("admin-key", "", "Admin key for API")
	db            *DB
	helper        plugins.PluginHelper
	WebPathPrefix = "webhook"
)

func main() {
	log := logger.CreateLogger(*Debug)
	log.Infof("Starting webhook plugin")
	err := envflag.Parse()
	if err != nil {
		log.Fatalf("Unable to load config: %s", err.Error())
		return
	}
	db, err = NewDB(*DBPath)
	if err != nil {
		log.Fatalf("Unable to load config: %s", err.Error())
		return
	}
	helper, err = plugins.NewHelper(*RPCHost, uint16(*RPCPort), *RPCToken)
	if err != nil {
		log.Fatalf("Unable to create helper: %s", err.Error())
		return
	}
	err = helper.RegisterWebhook(WebPathPrefix, handleWebhook)
	if err != nil {
		log.Fatalf("Unable to register webhook: %s", err.Error())
		return
	}
}

func checkAuth(request *rpc.HttpRequest) (bool, error) {
	for index := range request.Header {
		if strings.ToLower(request.Header[index].Key) == "x-api-key" {
			if request.Header[index].Value == *AdminKey {
				return true, nil
			}
			if db.CheckUser(request.Header[index].Value) {
				return false, nil
			}
		}
	}
	return false, errors.New("unauthorized")
}

func handleWebhook(request *rpc.HttpRequest) *rpc.HttpResponse {
	admin, err := checkAuth(request)
	if err != nil {
		return &rpc.HttpResponse{
			Body:   []byte(err.Error()),
			Status: http.StatusUnauthorized,
		}
	}
	path := strings.ToLower(request.Path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, WebPathPrefix)
	path = strings.TrimPrefix(path, "/")
	if admin {
		if strings.HasPrefix(path, "keys") {
			return handleAdminKeys(request)
		}
	}
	if strings.HasPrefix(path, "sendmessage") {
		return sendMessage(request)
	}
	return &rpc.HttpResponse{
		Body:   []byte("Unknown"),
		Status: http.StatusBadRequest,
	}
}

func handleAdminKeys(request *rpc.HttpRequest) *rpc.HttpResponse {
	switch request.Method {
	case "POST":
		body := &HookBody{}
		err := json.Unmarshal(request.Body, body)
		if err != nil {
			return &rpc.HttpResponse{
				Body:   []byte("Unable to decode"),
				Status: http.StatusInternalServerError,
			}
		}
		return addKey(body.Message)
	case "GET":
		return listKeys()
	case "DELETE":
		body := &HookBody{}
		err := json.Unmarshal(request.Body, body)
		if err != nil {
			return &rpc.HttpResponse{
				Body:   []byte("Unable to decode"),
				Status: http.StatusInternalServerError,
			}
		}
		return deleteKey(body.Message)
	default:
		return &rpc.HttpResponse{
			Body:   []byte("Unknown action"),
			Status: http.StatusBadRequest,
		}
	}
}

func listKeys() *rpc.HttpResponse {
	users := db.getUsers()
	userJson, err := json.Marshal(&users)
	if err != nil {
		return &rpc.HttpResponse{
			Body:   []byte("Unable to get keys"),
			Status: http.StatusInternalServerError,
		}
	}
	return &rpc.HttpResponse{
		Body: userJson,
		Header: []*rpc.HttpHeader{{
			Key:   "Content-Type",
			Value: "application/json",
		}},
		Status: http.StatusOK,
	}
}

func addKey(key string) *rpc.HttpResponse {
	users := db.getUsers()
	if len(users) > 0 {
		found := false
		for index := range users {
			if users[index] == key {
				found = true
				break
			}
		}
		if found {
			return &rpc.HttpResponse{
				Body:   []byte("User exists"),
				Status: http.StatusNoContent,
			}
		}
	}
	err := db.CreateUser(key)
	if err != nil {
		return &rpc.HttpResponse{
			Body:   []byte("Unable to get keys}"),
			Status: http.StatusInternalServerError,
		}
	}
	return &rpc.HttpResponse{
		Body:   []byte("User added"),
		Status: http.StatusOK,
	}
}

func deleteKey(key string) *rpc.HttpResponse {
	users := db.getUsers()
	found := false
	for index := range users {
		if users[index] == key {
			found = true
			break
		}
	}
	if !found {
		return &rpc.HttpResponse{
			Body:   []byte("User not found"),
			Status: http.StatusNotFound,
		}
	}
	err := db.DeleteUser(key)
	if err != nil {
		return &rpc.HttpResponse{
			Body:   []byte("Unable to get keys}"),
			Status: http.StatusInternalServerError,
		}
	}
	return &rpc.HttpResponse{
		Body:   []byte("User deleted"),
		Status: http.StatusOK,
	}
}

func sendMessage(request *rpc.HttpRequest) *rpc.HttpResponse {
	body := &HookBody{}
	err := json.Unmarshal(request.Body, body)
	if err != nil {
		return &rpc.HttpResponse{
			Body:   []byte("Unable to decode"),
			Status: http.StatusInternalServerError,
		}
	}
	errs := helper.SendIRCMessage(*Channel, []string{body.Message})
	if len(errs) > 0 {
		return &rpc.HttpResponse{
			Body:   []byte("Unable to send"),
			Status: http.StatusInternalServerError,
		}
	}
	return &rpc.HttpResponse{
		Body:   []byte("Delivered"),
		Status: http.StatusOK,
	}
}

type HookBody struct {
	Message string
}
