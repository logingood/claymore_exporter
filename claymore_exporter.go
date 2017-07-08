package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"net/http"
	"strings"
	"strconv"
	"flag"
	"encoding/json"
	"net/rpc/jsonrpc"

	"github.com/prometheus/client_golang/prometheus"
)

type ClaymoreStats struct {
	Uptime string `json:"uptime"`
	TotalRate string `json:"totalrate"`
	EthFound string `json:"ethfound"`
	HashRate []string 
}

type expConf struct {
	Dial_Addr []string 
	Port string 
	Proto string 
	Method string 
}

func fillDefaults() *expConf {
   confDefault := &expConf{
		 Dial_Addr: []string{"127.0.0.1"},
		 Port: "3333",
		 Proto: "tcp",
		 Method: "miner_getstat1",
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
		log.Fatal("Dialing:", err)
	}

	// Synchronous call
	c := jsonrpc.NewClient(client)
	err = c.Call(conf.Method, "", &reply)

	if err != nil {
		log.Fatal("Can't parse response:", err)
	}

	return reply
}

func parseReply(reply *json.RawMessage) *ClaymoreStats {

	j, err := json.Marshal(&reply)
	if err != nil {
		panic(err)
	}
	var result []string
	err = json.Unmarshal(j, &result)

	line1 := strings.Split(result[2], ";")
	line2 := strings.Split(result[3], ";")

	stats := &ClaymoreStats {
		Uptime: line1[1],
		TotalRate: line1[0],
		EthFound: result[1],
		HashRate: line2,
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
		[]string{"uptime", "Rig", "GPU"},
		nil)

	ethfoundDesc = prometheus.NewDesc(
		"eth_found",
		"Share count",
		[]string{"eth_found", "Rig", "GPU"},
		nil)

	totalrateDesc = prometheus.NewDesc(
		"total_hash_rate",
		"mh/s",
		[]string{"total_hash_rate", "Rig", "GPU"},
		nil)

	hashrateDesc = prometheus.NewDesc(
		"gpu_hash_rate",
		"kh/s",
		[]string{"gpu_hash_rate", "Rig", "GPU"},
		nil)
)


func (c *ClaymoreStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- uptimeDesc
	ch <- totalrateDesc
	ch <- ethfoundDesc
	ch <- hashrateDesc
}

func (c *ClaymoreStatsCollector) Collect(ch chan<- prometheus.Metric) {

		conf := readConf()
		for _, addr := range conf.Dial_Addr {

			reply := callClaymore(addr, conf)
			stats := parseReply(reply)

			uptime, _ := strconv.ParseFloat(stats.Uptime,32)
				
			ch <- prometheus.MustNewConstMetric(uptimeDesc,
				prometheus.GaugeValue,
				uptime,
				"Uptime", addr, "GPU_ALL")

			ethfound, _ := strconv.ParseFloat(stats.EthFound,32)
			ch <- prometheus.MustNewConstMetric(ethfoundDesc,
				prometheus.GaugeValue,
				ethfound,	
				"Ethfound", addr, "GPU_ALL")

			totalrate, _ := strconv.ParseFloat(stats.TotalRate,32)
			ch <- prometheus.MustNewConstMetric(totalrateDesc,
				prometheus.GaugeValue,
				totalrate,
				"TotalRate", addr, "GPU_ALL")

			var hashrate float64

			for i, val := range stats.HashRate {
				hashrate, _ = strconv.ParseFloat(val, 32)
				ch <- prometheus.MustNewConstMetric(hashrateDesc,
					prometheus.GaugeValue,
					hashrate,
					fmt.Sprintf("hash%d",i), addr,fmt.Sprintf("GPU%d",i))
			}
		}
		

}


func main() {

	var (
	  listenAddress  = flag.String("listen-address", ":10333", "Address on which to expose metrics and web interface.")
		metricsPath    = flag.String("telemetry-path", "/metrics", "Path under which to expose metrics.")
	)


  claymore_collector := NewClaymoreStatsCollector()

  prometheus.MustRegister(claymore_collector)

	http.Handle(*metricsPath, prometheus.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Sensor Exporter</title></head>
			<body>
			<h1>Claymore Stasts Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	http.ListenAndServe(*listenAddress, nil)


}
