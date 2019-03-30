package main

import (
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
)

var logger = logging.MustGetLogger("webhook")
var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} ▶ %{level} %{message}`,
)

type Config struct {
	ListenerPort int    `json:"listenerPort"`
	LogLevel     string `json:"logLevel"`
	ZabbixHost   string `json:"zabbixHost"`
}

type Problem struct {
	ProblemID string `json:"ProblemID"`
}

var config Config

func ZabbixHandler(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var problem Problem

	err := decoder.Decode(&problem)
	if err != nil {
		logger.Errorf("Could not parse the problem from the request body: %s", err.Error())
	}

	logger.Debugf("Parsed problem: %+v", problem)

	logger.Debug("Attempting to execute zabbix_sender")
	exec.Command("ls -ltrh")

	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Infof("IP: %s, Method: %s, URL: %s, Content-Lenght: %d", r.RemoteAddr, r.Method, r.RequestURI, r.ContentLength)
		next.ServeHTTP(w, r)
	})
}

func main() {

	// Set up logging
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)

	folderPath := filepath.Join(exPath, "log")
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

	// Read configuration
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

	logging.SetLevel(logging.DEBUG, "webhook")

	router := mux.NewRouter()
	router.HandleFunc("/zabbix", ZabbixHandler).Methods("POST")
	router.Use(loggingMiddleware)

	log.Fatal(http.ListenAndServe(":5000", router))
}

// Senha WIFI 192.168.15.1
// Contato vivo 7 dias = 99610 3054 Carlos Vivo
