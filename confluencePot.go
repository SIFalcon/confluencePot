/*
Copyright (C) <2022>  <SECUINFRA Falcon Team>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.
You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>
*/

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

var dirStr string

// check errors as they occur and panic :o
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// writeToFile saves the Log output
func writeToFile(filename string, data string) error {

	// create / open log file
	file, openErr := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	check(openErr)
	defer file.Close()

	// write log string into file
	_, writeStrErr := io.WriteString(file, data)
	check(writeStrErr)

	return file.Sync()
}

func logger(r *http.Request) {

	// dump the http request
	dumpedRequest, dumpErr := httputil.DumpRequest(r, true)
	check(dumpErr)

	// assemble log output
	logdt := time.Now()
	datestr := "[" + logdt.Format("01-02-2006_15:04:05") + "]\n"
	remoteAddr := "Remote address: " + r.RemoteAddr
	finalStr := datestr + remoteAddr + "\n" + string(dumpedRequest)
	color.Blue(datestr)
	color.Red(remoteAddr)
	fmt.Printf("%s", string(dumpedRequest))

	// log to file
	writeFileErr := writeToFile(dirStr, finalStr)
	check(writeFileErr)
}

func main() {

	fmt.Printf("  ___           __ _                      ___     _   \n")
	fmt.Printf(" / __|___ _ _  / _| |_  _ ___ _ _  __ ___| _ \\___| |_ \n")
	fmt.Printf("| (__/ _ \\ ' \\|  _| | || / -_) ' \\/ _/ -_)  _/ _ \\  _|\n")
	fmt.Printf(" \\___\\___/_||_|_| |_|\\_,_\\___|_||_\\__\\___|_| \\___/\\__|\n\n")

	// get current working directory for log file handling
	dir, gwdErr := os.Getwd()
	check(gwdErr)

	// get current timestamp for log file
	dt := time.Now()
	dtStr := dt.Format("01-02-2006_15:04:05")
	dirStr += dir + "/" + dtStr + ".log"
	fmt.Printf("Logging to %s\n\n", dirStr)

	// start new mux server
	mux := http.NewServeMux()

	// read fake confluence page
	siteBytes, readErr := ioutil.ReadFile("confluence.html")
	check(readErr)

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		// don't log favicon requests
		if !(strings.Contains(req.URL.Path, "favicon")) {
			logger(req)
		}

		// check if the requested URL contains "${" --> very likely exploitation attempt
		if strings.Contains(req.URL.RawPath, "%24%7B") || strings.Contains(req.URL.RawPath, "$%7B") {
			// return mock cmd response and HTTP status 302
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Header().Add("X-Cmd-Response", "lmao nice try")
			w.WriteHeader(302)
		} else {
			// return fake confluence site
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write(siteBytes)
		}
	})

	//set up TLS configuration
	// based on this implementation: https://gist.github.com/denji/12b3a568f092ab951456
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}
	// set up the HTTP server configuration
	srv := &http.Server{
		Addr:         ":443",
		Handler:      mux,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	log.Fatal(srv.ListenAndServeTLS("server.crt", "server.key"))
}
