package main

import (
	"encoding/json"
	"fmt"
	"github.com/jessevdk/go-flags"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type StatusResponse struct {
	Code    string
	Message string
	Details DetailsInfo
}

type DetailsInfo struct {
	Healthy bool
	Hostname string
	Token string
	IsActiveNode bool
	ActiveNode interface{}
	Err string
	AvailableNodes interface{}
	RaftLeader string
	IsRaftLeader bool
	RaftLeaderURI string
	RaftAdvertise string
	RaftHealthyMembers []string
}

type orchesExporterOpts struct {
	Host     string `short:"H" long:"host" default:"localhost" description:"Hostanme"`
	Port     string `short:"P" long:"port" default:"3000" description:"Port"`
	User     string `short:"U" long:"user" default:"admin" description:"User"`
	Password string `short:"p" long:"password" default:"admin" description:"Password"`
	Listen   string `short:"L" long:"listen" default:"localhost:1010" description:"Listen"`
}

func main() {
	logger := log.New(os.Stdout, "[Orchestrator_Exporter]", log.Lshortfile|log.Ldate|log.Ltime)
	opts := orchesExporterOpts{}
	psr := flags.NewParser(&opts, flags.Default)
	psr.Usage = "[OPTIONS]"
	_, err := psr.ParseArgs(os.Args)
	if err != nil {
		os.Exit(1)
	}

	//初始化一个http handler
	http.Handle("/metrics", promhttp.Handler())

	//初始化一个容器
	orchesStat := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "orches_status",
		Help: "orchestrator cluster health",
	},
		[]string{"status"},
	)
	prometheus.MustRegister(orchesStat)

	go func() {
		logger.Println("ListernAndServe:", opts.Listen)
		err := http.ListenAndServe(opts.Listen, nil)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()

	for {
		client := &http.Client{
			Transport: &http.Transport{Proxy: func(req *http.Request) (*url.URL, error) {
				req.SetBasicAuth(opts.User, opts.Password)
				return nil, nil
			},
			},
		}
		uri := fmt.Sprintf("http://%s:%s/api/health/", opts.Host, opts.Port)
		logger.Println(uri)
		resp, err := client.Get(uri)
		if err != nil {
			fmt.Println(err.Error())
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		var status StatusResponse
		err = json.Unmarshal(body, &status)
		if err != nil {
			logger.Println("jsonUnmarshal Error:", err.Error())
			os.Exit(2)
		}

		if status.Code == "OK" {
			if status.Details.Healthy {
				orchesStat.WithLabelValues("orchesStatus").Set(1)
			}else {
				orchesStat.WithLabelValues("orchesStatus").Set(0)
			}
		}else {
			logger.Println(status.Message)
		}

		time.Sleep(time.Duration(2) * time.Second)
	}
}
