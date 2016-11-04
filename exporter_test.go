package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const htmlResponse = "<!DOCTYPE html>\n<head>\n    <title>UW Documentation</title>\n</head>\n<body>\n<h1>UW Documented services</h1>\n<table style='font-size: 10pt; font-family: MONOSPACE;'>\n    \n    \n    <tr>\n        <td><a href=\"/../../../billing/services/uw-service-refdata:80/__/about\">billing.uw-service-refdata</a></td>\n    </tr>\n    \n    \n</table>\n</body>\n</html>"
const jsonResponse = "[{\"Service\":{\"Name\":\"uw-service-refdata\",\"Namespace\":\"billing\",\"BaseURL\":\"\"},\"Doc\":null}]"

func TestExporterService(t *testing.T) {
	errors := make(chan error, 10)
	about := make(chan About, 10)
	exporters := []exporter{newHTTPExporter(), newHTTPExporter()}
	e := exporterService{exporters: exporters}
	about <- About{}
	close(about)
	close(errors)
	e.export(about, errors)
	//give the exporters a chance to process as they run in different go routines
	time.Sleep(1 * time.Second)

	for i, ex := range e.exporters {
		fmt.Println(i)
		assert.Equal(t, 1, func() int {
			ex.(*httpExporter).mutex.RLock()
			l := len(ex.(*httpExporter).abouts)
			ex.(*httpExporter).mutex.RUnlock()
			return l
		}())
	}

}

func TestHTTPExporterHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name        string
		req         *http.Request
		exporter    *httpExporter
		statusCode  int
		contentType string // Contents of the Content-Type header
		body        string
	}{
		{"Success html", newRequest("GET", "/__/about", "text/html"), createHTTPExporterAndHandle(About{Service: Service{Name: "uw-service-refdata", Namespace: "billing"}}), http.StatusOK, "text/html", htmlResponse},
		{"Success json", newRequest("GET", "/__/about", "application/json"), createHTTPExporterAndHandle(About{Service: Service{Name: "uw-service-refdata", Namespace: "billing"}}), http.StatusOK, "application/json", jsonResponse},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(test.exporter).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(strings.TrimSpace(test.body), strings.TrimSpace(rec.Body.String()), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func createHTTPExporterAndHandle(about About) *httpExporter {
	httpExporter := newHTTPExporter()
	httpExporter.handle(about)
	return httpExporter
}

func newRequest(method, url string, accept string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	req.Header.Add("Accept", accept)
	if err != nil {
		panic(err)
	}
	return req
}

func router(e *httpExporter) *mux.Router {
	m := mux.NewRouter()
	m.HandleFunc("/__/about", e.handleHTTP).Methods("GET")
	return m
}
