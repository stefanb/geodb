package metrics

import (
	api "github.com/autom8ter/geodb/gen/go/geodb"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(objectLat, objectLon)
}

var (
	objectLat = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "object_latitude",
		Help: "the objects latitude",
	}, []string{"key"})
	objectLon = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "object_longitude",
		Help: "the objects longitude",
	}, []string{"key"})
)

func GaugeObjectLocation(key string, point *api.Point) {
	objectLat.WithLabelValues(key).Set(point.Lat)
	objectLon.WithLabelValues(key).Set(point.Lon)
}
