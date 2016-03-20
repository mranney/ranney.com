package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
)

type redirHandler struct {
}

func (h *redirHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("method=%s url=%s proto=%d.%d host=%s remote=%s code=%d\n",
		r.Method, r.URL.String(), r.ProtoMajor, r.ProtoMinor, r.Host, r.RemoteAddr, http.StatusMovedPermanently)
	newUrl := fmt.Sprintf("https://%s%s", r.Host, r.URL.String())
	http.Redirect(w, r, newUrl, http.StatusMovedPermanently)
}

type responseStats struct {
	StatusCode    int
	ResponseBytes int
}

type logResponseWriter struct {
	http.ResponseWriter
	Res *responseStats
}

func (w logResponseWriter) WriteHeader(statusCode int) {
	w.Res.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w logResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w logResponseWriter) Write(b []byte) (int, error) {
	w.Res.ResponseBytes += len(b)
	return w.ResponseWriter.Write(b)
}

type loggingHandler struct {
}

func (h *loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "ranney.com")

	if strings.HasPrefix(r.URL.Path, "/~mjr") {
		r.URL.Path = strings.Replace(r.URL.Path, "/~mjr", "/mjr", 1)
	}

	resStats := responseStats{}
	lrw := logResponseWriter{w, &resStats}

	handler := http.FileServer(http.Dir("/home/freebsd/ranney.com/files"))
	handler.ServeHTTP(lrw, r)

	var versionStr string
	switch r.TLS.Version {
	case tls.VersionSSL30:
		versionStr = "SSL3.0"
	case tls.VersionTLS10:
		versionStr = "TLS1.0"
	case tls.VersionTLS11:
		versionStr = "TLS1.1"
	case tls.VersionTLS12:
		versionStr = "TLS1.2"
	}

	tlsStr := fmt.Sprintf("tls=true version=%s server_name=%s", versionStr, r.TLS.ServerName)

	refStr := r.Header.Get("Referer")

	fmt.Printf("method=%s url=%s proto=%d.%d host=%s remote=%s code=%d bytes=%d ref=%s %s\n",
		r.Method, r.URL.String(), r.ProtoMajor, r.ProtoMinor, r.Host, r.RemoteAddr,
		resStats.StatusCode, resStats.ResponseBytes, refStr, tlsStr)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
}

func listenRedir() {
	fmt.Println("Listening on HTTP, port 80")
	err := http.ListenAndServe(":80", &redirHandler{})
	log.Fatal(err)
}

func main() {
	go listenRedir()

	fmt.Println("Listening on HTTPS, port 443")
	err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/ranney.com/fullchain.pem", "/etc/letsencrypt/live/ranney.com/privkey.pem", &loggingHandler{})
	log.Fatal(err)
}
