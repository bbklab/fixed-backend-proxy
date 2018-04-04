package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	prefix = "/api/dmos/" // must with suffix /
)

func main() {
	var (
		https, _ = strconv.ParseBool(os.Getenv("BACKEND_HTTPS"))
		backend  = os.Getenv("BACKEND_ENDPOINT")
		listen   = os.Getenv("LISTEN")
	)

	if backend == "" {
		log.Fatalln("env BACKEND_ENDPOINT required")
	}

	if listen == "" {
		listen = ":8080"
	}

	_, _, err := net.SplitHostPort(backend)
	if err != nil {
		log.Fatalln(err)
	}

	if https {
		isHttps, err := detectHTTPs(backend)
		if err != nil {
			log.Fatalln("detect https backend error:", err)
		}
		if !isHttps {
			log.Fatalf("backend %s seems not serving https", backend)
		}
	}

	http.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
		// hijack client underlying net.Conn
		src, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			http.Error(w, fmt.Sprintf("hijack client error: %v", err), 500)
			return
		}
		defer src.Close()

		// dial backend
		dst, err := net.DialTimeout("tcp", backend, time.Second*10)
		if err != nil {
			errmsg := fmt.Sprintf("dial backend error: %v", err)
			src.Write([]byte("HTTP/1.0 500 Internal Server Error\r\n\r\n" + errmsg + "\r\n"))
			return
		}
		defer dst.Close()

		if https {
			dst, err = wrapWithTLS(dst)
			if err != nil {
				errmsg := fmt.Sprintf("tls handshake with backend error: %v", err)
				src.Write([]byte("HTTP/1.0 500 Internal Server Error\r\n\r\n" + errmsg + "\r\n"))
				return
			}
		}

		// rewrite client request
		r.URL.Host = backend
		r.URL.Path = r.URL.Path[len(prefix)-1:]
		r.RequestURI = ""

		// send origin request
		if err := r.WriteProxy(dst); err != nil {
			errmsg := fmt.Sprintf("write request to backend error: %v", err)
			src.Write([]byte("HTTP/1.0 500 Internal Server Error\r\n\r\n" + errmsg + "\r\n"))
			return
		}

		// log
		var (
			srcAddr = src.RemoteAddr().String()
			dstAddr = dst.RemoteAddr().String()
			method  = r.Method
			path    = r.URL.Path
		)
		log.Println(method, path, srcAddr, "<-->", dstAddr)

		// io copy between src & dst
		go func() {
			io.Copy(dst, src)
			dst.Close()
			// fmt.Println("A quit")
		}()
		io.Copy(src, dst) // note: hanging wait while copying the response until EOF
		//fmt.Println("B quit")
	})

	log.Fatal(http.ListenAndServe(listen, nil))
}

// wrap a plain net.Conn with tls and try tls handshake
func wrapWithTLS(plainConn net.Conn) (net.Conn, error) {
	tlsConn := tls.Client(plainConn, &tls.Config{InsecureSkipVerify: true})

	errCh := make(chan error, 2)
	timer := time.AfterFunc(time.Second*10, func() {
		errCh <- errors.New("timeout on tls handshake")
	})
	defer timer.Stop()

	go func() {
		errCh <- tlsConn.Handshake()
	}()

	if err := <-errCh; err != nil {
		return nil, err
	}
	return tlsConn, nil
}

func detectHTTPs(addr string) (https bool, err error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second*60)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	if err != nil {
		return
	}

	b := make([]byte, 5)
	_, err = conn.Read(b)
	if err != nil {
		return
	}

	https = string(b[:]) != "HTTP/" // or use: b[0] == 21
	return
}
