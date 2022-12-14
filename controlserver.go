package sqlconf

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	//"fmt"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
)

func GetClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if ip != "" {
		return ip
	}

	ip = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	if ip != "" {
		return ip
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}

func IsAnyContinue(w http.ResponseWriter, r *http.Request) bool {
	cip := GetClientIP(r)
	isa := IsAllow(cip)
	zapLogger.Info("allow/block", zap.String("client-ip", cip), zap.Bool("is-allow", isa))
	if isa != true {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(cip + ", " + r.Proto + " ,you cannot visit this site."))
		return false
	}
	return true
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	cip := GetClientIP(r)

	if r.URL.Path == "/" {
		w.WriteHeader(http.StatusOK)

		w.Write([]byte("welcome<br/>"))
		w.Write([]byte(cip + "<br/>"))
		w.Write([]byte(r.Proto))
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	}

}

func RemoteShutdownHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	w.WriteHeader(http.StatusGone)
	w.Write([]byte("shutdown the server in 3 seconds..."))
	go func() {
		time.Sleep(3 * time.Second)
		zapLogger.Info("app will exit after 3 seconds")
		os.Exit(0)
	}()

}

func EchoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hi"))

}

type anyHandler struct {
}

func (ah anyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cip := GetClientIP(r)
	isa := IsAllow(cip)
	if isa != true {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(cip + ", " + r.Proto + " ,you cannot visit this site."))
		return
	}
	if r.URL.Path == "/" {
		IndexHandler(w, r)
	}

	if r.URL.Path == "/remote-shutdown" {
		RemoteShutdownHandler(w, r)
	}

	if r.URL.Path == "/echo" {
		EchoHandler(w, r)
	}

}

func (h2s *H2Server) runControlServer() {
	addr := strings.Join([]string{h2s.IP, strconv.Itoa(h2s.Port + 1)}, ":")

	controlServer := http.Server{
		Addr:    addr,
		Handler: &anyHandler{},
	}

	zapLogger.Info("runControlServer", zap.String("address", addr))
	http2.ConfigureServer(&controlServer, &http2.Server{})

	zapLogger.Info("runControlServer", zap.String("address", addr))
	err := controlServer.ListenAndServeTLS(h2s.TLScert, h2s.TLSkey)
	if err != nil {
		zapLogger.Error("runControlServer", zap.Error(err))
	}
}
