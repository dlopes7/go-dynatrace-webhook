package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/op/go-logging"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// TODO: Add proper problem properties
// TODO: Add key selection on Post Body?

var logger = logging.MustGetLogger("webhook")
var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} ▶ %{level} %{message}`,
)

type Config struct {
	ListenerPort         int    `json:"listenerPort"`
	LogLevel             string `json:"logLevel"`
	ZabbixHost           string `json:"zabbixHost"`
	ZabbixServerHostname string `json:"zabbixServerHostname"`
	ZabbixServerPort     int    `json:"zabbixServerPort"`
	ZabbixItem           string `json:"zabbixItem"`
}

type Problem struct {
	ProblemID          string `json:"ProblemID"`
	State              string `json:"State"`
	ProblemDetailsText string `json:"ProblemDetailsText"`
	ProblemTitle       string `json:"ProblemTitle"`
}

func (p Problem) String() string {
	return fmt.Sprintf("ProblemID: %s, State: %s, Title: %s, Details: %s", p.ProblemID, p.State, p.ProblemTitle, p.ProblemDetailsText)
}

type Response struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

var config Config

func ZabbixHandler(w http.ResponseWriter, r *http.Request) {

	resp := Response{}

	decoder := json.NewDecoder(r.Body)
	var problem Problem

	err := decoder.Decode(&problem)
	if err != nil {
		logger.Errorf("Could not parse the problem from the request body: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		resp = Response{
			Error:   true,
			Message: fmt.Sprintf("Could not parse the problem from the request body: %s", err.Error()),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	logger.Debugf("Parsed problem: %+v", problem)

	var args = []string{
		"-z", config.ZabbixServerHostname,
		"-p", strconv.Itoa(config.ZabbixServerPort),
		"-s", config.ZabbixHost,
		"-k", config.ZabbixItem,
		"-o", fmt.Sprintf("\"%s\"", problem.String()),
		"-vv"}

	logger.Debugf("Attempting to execute command: 'zabbix_sender %s'", strings.Join(args, " "))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "zabbix_sender", args...)

	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		logger.Errorf("Timeout after 15 seconds executing zabbix_sender")
		w.WriteHeader(http.StatusInternalServerError)
		resp = Response{
			Error:   true,
			Message: "Timed out after 15 seconds waiting for zabbix_sender",
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	if err != nil {
		logger.Errorf("Error executing zabbix_sender: %s Result: %s", err.Error(), out)
		w.WriteHeader(http.StatusInternalServerError)
		resp = Response{
			Error:   true,
			Message: fmt.Sprintf("Error executing zabbix_sender: %s", err.Error()),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	logger.Debugf("Command result: %s", out)
	resp = Response{
		Error:   false,
		Message: fmt.Sprintf("Zabbix Sender executed successfully: %s", out),
	}

	logger.Infof("Request processed successfully: %+v", resp.Message)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Infof("IP: %s, Method: %s, URL: %s, Content-Lenght: %d", r.RemoteAddr, r.Method, r.RequestURI, r.ContentLength)
		next.ServeHTTP(w, r)
	})
}

func main() {

	log.Println("Setting up logging...")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)

	folderPath := filepath.Join(exPath, "../log")
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		panic(err)
	}

	logPath := filepath.Join(folderPath, "webhook.log")

	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file: %v", err)
	}

	defer lf.Close()
	logBackend := logging.NewLogBackend(lf, "", 0)
	logging.SetFormatter(format)
	logging.SetBackend(logBackend)
	log.Printf("Logging to '%s'", logPath)

	log.Println("Reading config.json...")
	configFile, err := os.Open("config.json")
	if err != nil {
		logger.Fatalf("Could not read config.json: %s", err.Error())
	}
	defer configFile.Close()

	byteValue, _ := ioutil.ReadAll(configFile)
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		logger.Fatalf("Could not parse the configuration file: %s", err.Error())
	}

	log.Printf("Config read: %+v", config)

	logLevel, err := logging.LogLevel(config.LogLevel)
	if err != nil {
		logger.Fatalf("Invalid log level %s, options are CRITICAL, ERROR, WARNING, INFO, DEBUG", config.LogLevel)
	}
	logging.SetLevel(logLevel, "webhook")
	logger.Infof("Server will start with config %+v", config)

	log.Println("Setting up routes...")
	router := mux.NewRouter()
	router.HandleFunc("/zabbix", ZabbixHandler).Methods("POST")
	router.Use(loggingMiddleware)

	log.Printf("Server started at port %d\n", config.ListenerPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.ListenerPort), router))

}
