package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/greboid/irc/v2/logger"
	"github.com/greboid/irc/v2/rpc"
	"github.com/kouhin/envflag"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
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
	WebPathPrefix = "webhook"
)

type webPlugin struct {
	db       *DB
	adminKey string
	RPCConn  *grpc.ClientConn
	Channel  string
	log      *zap.SugaredLogger
}

func main() {
	log := logger.CreateLogger(*Debug)
	log.Infof("Starting webhook plugin")
	if err := envflag.Parse(); err != nil {
		log.Fatalf("Unable to load config: %s", err.Error())
		return
	}
	db, err := NewDB(*DBPath)
	if err != nil {
		log.Fatalf("Unable to load config: %s", err.Error())
		return
	}
	creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", *RPCHost, *RPCPort), grpc.WithTransportCredentials(creds))
	defer func() { _ = conn.Close() }()
	if err != nil {
		log.Panicf("Unable to load create RPC: %s", err.Error())
	}
	plugin := webPlugin{
		db:       db,
		adminKey: *AdminKey,
		RPCConn:  conn,
		log:      log,
	}
	plugin.run()
}

func (p *webPlugin) run() {
	client := rpc.NewHTTPPluginClient(p.RPCConn)
	stream, err := client.GetRequest(rpc.CtxWithTokenAndPath(context.Background(), "bearer", *RPCToken, WebPathPrefix))
	if err != nil {
		p.log.Errorf("Unable to connect to RPC")
		return
	}
	for {
		request, err := stream.Recv()
		if err == io.EOF {
			p.log.Debugf("RPC ended.")
			return
		}
		if err != nil {
			p.log.Errorf("Error talking to RPC: %s", err.Error())
			return
		}
		response := p.handleWebhook(request)
		err = stream.Send(response)
		if err != nil {
			p.log.Errorf("Error sending response: %s", err.Error())
			continue
		}
	}
}

func (p *webPlugin) checkAuth(request *rpc.HttpRequest) (bool, error) {
	for index := range request.Header {
		if strings.ToLower(request.Header[index].Key) == "x-api-key" {
			if request.Header[index].Value == p.adminKey {
				return true, nil
			}
			if p.db.CheckUser(request.Header[index].Value) {
				return false, nil
			}
		}
	}
	return false, errors.New("unauthorized")
}

func (p *webPlugin) handleWebhook(request *rpc.HttpRequest) *rpc.HttpResponse {
	client := rpc.NewIRCPluginClient(p.RPCConn)
	admin, err := p.checkAuth(request)
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
			return p.handleAdminKeys(request)
		}
	}
	if strings.HasPrefix(path, "sendmessage") {
		return p.sendMessage(request, client)
	}
	return &rpc.HttpResponse{
		Body:   []byte("Unknown"),
		Status: http.StatusBadRequest,
	}
}

func (p *webPlugin) handleAdminKeys(request *rpc.HttpRequest) *rpc.HttpResponse {
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
		return p.addKey(body.Message)
	case "GET":
		return p.listKeys()
	case "DELETE":
		body := &HookBody{}
		err := json.Unmarshal(request.Body, body)
		if err != nil {
			return &rpc.HttpResponse{
				Body:   []byte("Unable to decode"),
				Status: http.StatusInternalServerError,
			}
		}
		return p.deleteKey(body.Message)
	default:
		return &rpc.HttpResponse{
			Body:   []byte("Unknown action"),
			Status: http.StatusBadRequest,
		}
	}
}

func (p *webPlugin) listKeys() *rpc.HttpResponse {
	users := p.db.getUsers()
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

func (p *webPlugin) addKey(key string) *rpc.HttpResponse {
	users := p.db.getUsers()
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
	err := p.db.CreateUser(key)
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

func (p *webPlugin) deleteKey(key string) *rpc.HttpResponse {
	users := p.db.getUsers()
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
	err := p.db.DeleteUser(key)
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

func (p *webPlugin) sendMessage(request *rpc.HttpRequest, client rpc.IRCPluginClient) *rpc.HttpResponse {
	body := &HookBody{}
	err := json.Unmarshal(request.Body, body)
	if err != nil {
		return &rpc.HttpResponse{
			Body:   []byte("Unable to decode"),
			Status: http.StatusInternalServerError,
		}
	}
	_, err = client.SendChannelMessage(rpc.CtxWithToken(context.Background(), "bearer", *RPCToken), &rpc.ChannelMessage{
		Channel: *Channel,
		Message: body.Message,
	})
	if err != nil {
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
