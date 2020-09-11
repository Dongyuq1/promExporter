package metrics

import (
	"ccm/exporter/mongodb"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var MetricsNameLatency = "CCM_Latency"
var MetricsHelpLatency = "(latency of services accessing gateway)"

type CCMMetrics struct {
	MetricsDescs []*prometheus.Desc
}

type CCMMetricInfo struct {
	MetricName  string
	MetricsHelp string
	MetricsType string
	ValueType   string
}

func (c *CCMMetrics) Describe(ch chan<- *prometheus.Desc) {
	len1 := len(c.MetricsDescs)
	for i := 0; i < len1; i++ {
		ch <- c.MetricsDescs[i]
	}
}

func (c *CCMMetrics) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	nowUTC := start.UTC()
	resp := mongodb.QueryMtrAllData(nowUTC.Unix())
	for _, v := range resp.MtrDataByISP {
		isp := v.ISP
		fmt.Println("......................", isp)
		ts := time.Unix(v.Clientutc, 0)
		for _, v2 := range v.MtrDataByGW {
			tmpLatency := v2.Latency
			gateway := v2.ServerLocation
			domain := v2.Url
			tmp := prometheus.NewDesc(
				MetricsNameLatency,
				MetricsHelpLatency,
				[]string{"Name"},
				prometheus.Labels{"gateway": gateway, "isp": isp, "domain": domain},
			)
			ch <- prometheus.NewMetricWithTimestamp(
				ts,
				prometheus.MustNewConstMetric(
					tmp,
					prometheus.GaugeValue,
					tmpLatency,
					MetricsNameLatency,
				),
			)
		}
	}
	eT := time.Since(start)
	fmt.Printf("CCM Metrics, Elapsed Time: %s, Date(UTC): %s\n", eT, start.UTC().Format("2006/01/02T15:04:05"))
}

func AddLatency() *CCMMetrics {
	var tmpMetricsDescs []*prometheus.Desc
	resp := mongodb.QueryMtrAllData(time.Now().UTC().Unix())
	for _, v := range resp.MtrDataByISP {
		isp := v.ISP
		for _, v2 := range v.MtrDataByGW {

			gateway := v2.ServerLocation
			domain := v2.Url
			tmp := prometheus.NewDesc(
				MetricsNameLatency,
				MetricsHelpLatency,
				[]string{"Name"},
				prometheus.Labels{"gateway": gateway, "isp": isp, "domain": domain},
			)
			tmpMetricsDescs = append(tmpMetricsDescs, tmp)
		}
	} //aws

	api := &CCMMetrics{MetricsDescs: tmpMetricsDescs}
	return api
}
