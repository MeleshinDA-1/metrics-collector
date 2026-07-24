package handler

import (
	"compress/gzip"
	"log/slog"
	"net/http"
	"strings"
)

type CompressingResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

func (w *CompressingResponseWriter) enableGzip() {
	if w.gzipWriter != nil {
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Del("Content-Length")
	w.gzipWriter = gzip.NewWriter(w.ResponseWriter)
}

func (w *CompressingResponseWriter) Write(data []byte) (int, error) {
	contentType := w.Header().Get("Content-Type")

	if IsCompressionAllowedForType(contentType) {
		w.enableGzip()
		return w.gzipWriter.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

func (w *CompressingResponseWriter) WriteHeader(statusCode int) {
	contentType := w.Header().Get("Content-Type")
	if IsCompressionAllowedForType(contentType) {
		w.enableGzip()
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *CompressingResponseWriter) Close() error {
	if w.gzipWriter == nil {
		return nil
	}

	return w.gzipWriter.Close()
}

func CompressingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				slog.Error("unable to create gzip reader", "error", err)
				http.Error(w, "invalid gzip body", http.StatusBadRequest)
				return
			}
			defer reader.Close()

			r.Body = reader
		}

		responseWriter := w
		if CanClientAcceptEncoding(r) {
			compressingWriter := &CompressingResponseWriter{
				ResponseWriter: w,
			}
			defer func() {
				if err := compressingWriter.Close(); err != nil {
					slog.Error("unable to close gzip writer", "error", err)
				}
			}()

			responseWriter = compressingWriter
		}

		next.ServeHTTP(responseWriter, r)
	})
}

func CanClientAcceptEncoding(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func IsCompressionAllowedForType(contentType string) bool {
	return strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html")
}
