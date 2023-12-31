package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}

type (
	responceData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responceData *responceData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responceData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responceData.status = statusCode
}

func WithLog(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responceData := &responceData{
			status: 0,
			size:   0,
		}

		lw := loggingResponseWriter{
			ResponseWriter: w,
			responceData:   responceData,
		}

		uri := r.RequestURI

		method := r.Method

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		Log.Info("Request: ",
			zap.String("method", method),
			zap.String("URL", uri),
			zap.String("duration", duration.String()))

		Log.Info("Response: ",
			zap.Int("Status", responceData.status),
			zap.Int("Size", responceData.size))
	})
}
