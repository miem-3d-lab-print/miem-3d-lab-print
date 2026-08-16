package metrics

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

type Metrics struct {
	registry     *prometheus.Registry
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
	sqlDuration  *prometheus.HistogramVec
	sqlErrors    *prometheus.CounterVec
}

func New(service string, sqlDB *sql.DB) *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		registry: registry,
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "app_http_requests_total", ConstLabels: prometheus.Labels{"service": service},
		}, []string{"method", "route", "status"}),
		httpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "app_http_request_duration_seconds", ConstLabels: prometheus.Labels{"service": service},
		}, []string{"method", "route"}),
		sqlDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "app_sql_query_duration_seconds", ConstLabels: prometheus.Labels{"service": service},
		}, []string{"operation"}),
		sqlErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "app_sql_errors_total", ConstLabels: prometheus.Labels{"service": service},
		}, []string{"operation"}),
	}
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewDBStatsCollector(sqlDB, service),
		metrics.httpRequests,
		metrics.httpDuration,
		metrics.sqlDuration,
		metrics.sqlErrors,
	)
	return metrics
}

func (metrics *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{})
}

func (metrics *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		response.Header().Set("X-Request-ID", requestID)

		started := time.Now()
		writer := &statusWriter{ResponseWriter: response, status: http.StatusOK}
		next.ServeHTTP(writer, request)

		route := request.Pattern
		if route == "" {
			route = "unmatched"
		}
		metrics.httpRequests.WithLabelValues(request.Method, route, strconv.Itoa(writer.status)).Inc()
		metrics.httpDuration.WithLabelValues(request.Method, route).Observe(time.Since(started).Seconds())
	})
}

func (metrics *Metrics) InstrumentGORM(database *gorm.DB) error {
	before := func(transaction *gorm.DB) {
		transaction.InstanceSet("metrics_started", time.Now())
	}
	after := func(operation string) func(*gorm.DB) {
		return func(transaction *gorm.DB) {
			if started, found := transaction.InstanceGet("metrics_started"); found {
				metrics.sqlDuration.WithLabelValues(operation).Observe(time.Since(started.(time.Time)).Seconds())
			}
			if transaction.Error != nil {
				metrics.sqlErrors.WithLabelValues(operation).Inc()
			}
		}
	}

	callbacks := []struct {
		name   string
		before func() error
		after  func() error
	}{
		{name: "create", before: func() error {
			return database.Callback().Create().Before("gorm:create").Register("metrics:before_create", before)
		}, after: func() error {
			return database.Callback().Create().After("gorm:create").Register("metrics:after_create", after("create"))
		}},
		{name: "query", before: func() error {
			return database.Callback().Query().Before("gorm:query").Register("metrics:before_query", before)
		}, after: func() error {
			return database.Callback().Query().After("gorm:query").Register("metrics:after_query", after("query"))
		}},
		{name: "update", before: func() error {
			return database.Callback().Update().Before("gorm:update").Register("metrics:before_update", before)
		}, after: func() error {
			return database.Callback().Update().After("gorm:update").Register("metrics:after_update", after("update"))
		}},
		{name: "delete", before: func() error {
			return database.Callback().Delete().Before("gorm:delete").Register("metrics:before_delete", before)
		}, after: func() error {
			return database.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", after("delete"))
		}},
	}
	for _, callback := range callbacks {
		if err := callback.before(); err != nil {
			return err
		}
		if err := callback.after(); err != nil {
			return err
		}
	}
	return nil
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (writer *statusWriter) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.status = status
	writer.wroteHeader = true
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *statusWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}
