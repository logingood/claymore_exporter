package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type ClaymoreStats struct {
	Uptime    string    `json:"uptime"`
	TotalRate string    `json:"totalrate"`
	EthFound  string    `json:"ethfound"`
	EthReject string    `json:"ethreject"`
	GPUs      []GPUInfo `json:"gpuinfo`
}

type GPUInfo struct {
	Name     string
	HashRate string
	Temp     string
	FanSpeed string
}

type expConf struct {
	Dial_Addr []string
	Port      string
	Proto     string
	Method    string
}

func fillDefaults() *expConf {
	confDefault := &expConf{
		Dial_Addr: []string{"127.0.0.1"},
		Port:      "3333",
		Proto:     "tcp",
		Method:    "miner_getstat1",
	}
	return confDefault
}

func readConf() *expConf {
	conf := fillDefaults()

	dial_addr := os.Getenv("CLAYMORE_DIAL_ADDR")
	if len(dial_addr) == 0 {
		panic("DIAL_ADDR env must be set, e.g.: export CLAYMORE_DIAL_ADDR=192.168.1.1;192.168.1.2;..")
	}

	dial_addr_slice := strings.Split(dial_addr, ";")
	conf.Dial_Addr = dial_addr_slice

	port := os.Getenv("CLAYMORE_PORT")
	if len(port) != 0 {
		conf.Port = port
	}

	proto := os.Getenv("CLAYMORE_PROTO")
	if len(proto) != 0 {
		conf.Proto = proto
	}

	method := os.Getenv("CLAYMORE_STATS")
	if len(method) != 0 {
		conf.Method = method
	}

	return conf
}

func callClaymore(addr string, conf *expConf) (reply *json.RawMessage) {

	client, err := net.Dial(conf.Proto, fmt.Sprintf("%s:%s", addr, conf.Port))

	if err != nil {
		log.Print("Dialing:", err)
		fake_reply := json.RawMessage(`["Fake Version", "0","0;0;0","0", "0;0;0", 
		"off;off;off;off", "0;0", "fake.miner", "0;0;0;0"]`)
		return &fake_reply
	} else {

		// Synchronous call
		c := jsonrpc.NewClient(client)
		err = c.Call(conf.Method, "", &reply)

		if err != nil {
			log.Fatal("Can't parse response:", err)
		}

		return reply
	}
}

func parseReply(reply *json.RawMessage) *ClaymoreStats {
	var temps []string
	var fans []string
	var result []string

	j, err := json.Marshal(&reply)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(j, &result)

	totals := strings.Split(result[2], ";")
	hashrate := strings.Split(result[3], ";")

	for i, v := range strings.Split(result[6], ";") {
		if i%2 == 0 {
			temps = append(temps, v)
		} else {
			fans = append(fans, v)
		}
	}

	GPUs := make([]GPUInfo, len(hashrate))
	for i := range GPUs {
		GPUs[i].FanSpeed = fans[i]
		GPUs[i].Temp = temps[i]
		GPUs[i].HashRate = hashrate[i]
		GPUs[i].Name = fmt.Sprintf("GPU%v", i)
	}

	// result[1] contains uptime of the miner
	// result[2] contains totals TotalHashRate;SharesFound;SharesRejected
	// result[3] contais  per-GPU hashrate

	stats := &ClaymoreStats{
		Uptime:    result[1],
		TotalRate: totals[0],
		EthFound:  totals[1],
		EthReject: totals[2],
		GPUs:      GPUs,
	}

	return stats
}

type ClaymoreStatsCollector struct{}

func NewClaymoreStatsCollector() *ClaymoreStatsCollector {
	return &ClaymoreStatsCollector{}
}

var (
	uptimeDesc = prometheus.NewDesc(
		"miner_total_uptime",
		"Minutes",
		[]string{"Rig"},
		nil)

	ethfoundDesc = prometheus.NewDesc(
		"eth_found",
		"Share count",
		[]string{"Rig"},
		nil)

	ethrejectDesc = prometheus.NewDesc(
		"eth_reject",
		"Rejected shares count",
		[]string{"Rig"},
		nil)

	totalrateDesc = prometheus.NewDesc(
		"total_hash_rate",
		"mh/s",
		[]string{"Rig"},
		nil)

	hashrateDesc = prometheus.NewDesc(
		"gpu_hash_rate",
		"kh/s",
		[]string{"Rig", "GPU"},
		nil)

	tempDesc = prometheus.NewDesc(
		"gpu_temp_celsius",
		"c",
		[]string{"Rig", "GPU"},
		nil)

	fanspeedDesc = prometheus.NewDesc(
		"gpu_fanspeed_percentage",
		"%",
		[]string{"Rig", "GPU"},
		nil)
)

func (c *ClaymoreStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- uptimeDesc
	ch <- totalrateDesc
	ch <- ethfoundDesc
	ch <- ethrejectDesc
	ch <- hashrateDesc
}

func (c *ClaymoreStatsCollector) Collect(ch chan<- prometheus.Metric) {

	conf := readConf()
	for _, addr := range conf.Dial_Addr {

		reply := callClaymore(addr, conf)
		stats := parseReply(reply)

		uptime, _ := strconv.ParseFloat(stats.Uptime, 32)

		ch <- prometheus.MustNewConstMetric(uptimeDesc,
			prometheus.GaugeValue,
			uptime,
			addr)

		ethfound, _ := strconv.ParseFloat(stats.EthFound, 32)
		ch <- prometheus.MustNewConstMetric(ethfoundDesc,
			prometheus.GaugeValue,
			ethfound,
			addr)

		ethreject, _ := strconv.ParseFloat(stats.EthReject, 32)
		ch <- prometheus.MustNewConstMetric(ethrejectDesc,
			prometheus.GaugeValue,
			ethreject,
			addr)

		totalrate, _ := strconv.ParseFloat(stats.TotalRate, 32)
		ch <- prometheus.MustNewConstMetric(totalrateDesc,
			prometheus.GaugeValue,
			totalrate,
			addr)

		for _, val := range stats.GPUs {
			hashrate, _ := strconv.ParseFloat(val.HashRate, 32)
			ch <- prometheus.MustNewConstMetric(hashrateDesc,
				prometheus.GaugeValue,
				hashrate,
				addr, val.Name)
		}

		for _, val := range stats.GPUs {
			temp, _ := strconv.ParseFloat(val.Temp, 32)
			ch <- prometheus.MustNewConstMetric(tempDesc,
				prometheus.GaugeValue,
				temp,
				addr, val.Name)
		}

		for _, val := range stats.GPUs {
			fanSpeed, _ := strconv.ParseFloat(val.FanSpeed, 32)
			ch <- prometheus.MustNewConstMetric(fanspeedDesc,
				prometheus.GaugeValue,
				fanSpeed,
				addr, val.Name)
		}
	}

}

func main() {

	var (
		listenAddress = flag.String("web.listen-address", ":10333", "Address on which to expose metrics and web interface.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	)

	claymore_collector := NewClaymoreStatsCollector()

	prometheus.MustRegister(claymore_collector)

	http.Handle(*metricsPath, prometheus.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Claymore Stats Exporter</title></head>
			<body>
			<h1>Claymore Stasts Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	http.ListenAndServe(*listenAddress, nil)

}
