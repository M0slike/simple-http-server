package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ldez/mimetype"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
)

type basicAuth struct {
	username string
	password string
	ok       bool
}
type Request struct {
	Method                  string
	Path                    string
	RequestUri              string
	Protocol                string
	Host                    string
	RemoteAddress           string
	ContentLength           int64
	ContentType             string
	Headers                 map[string][]string
	QueryParams             map[string][]string
	BasicAuth               *basicAuth
	BodyIsString            bool
	BodyParseError          error
	Body                    []byte
	BodyFormValues          map[string][]string
	BodyMultipartFormValues *multipart.Form
}

func (r *Request) printHeaders() {
	if len(r.Headers) > 0 {
		fmt.Println("Headers:")
		printStrSliceMap(r.Headers)
	}
}

func (r *Request) printAuth() {
	if r.BasicAuth.ok {
		auth := make(map[string]string)
		auth["type"] = "Basic"
		auth["username"] = r.BasicAuth.username
		auth["password"] = r.BasicAuth.password

		fmt.Println("Authorization:")
		printStrMap(auth)
	}
}

func (r *Request) printQueryParams() {
	if len(r.QueryParams) > 0 {
		fmt.Println("Query params:")
		printStrSliceMap(r.QueryParams)
	}
}

func (r *Request) printBody() {
	if len(r.Body) == 0 {
		return
	}

	if r.BodyIsString {
		if strings.Contains(r.ContentType, "json") {
			if json.Valid(r.Body) {
				fmt.Println("Body (valid json):")

				if Cfg.ShouldFormatJson {
					err := printJsonIndented(r.Body)
					if err != nil {
						log.Printf("error indenting json body: %v", err)
						printBytes(r.Body)
					}
				}
			} else {
				fmt.Println("Body (invalid json):")
				printBytes(r.Body)
			}

			return
		}

		fmt.Println("Body:")
		printBytes(r.Body)
		return
	}

	if r.BodyFormValues != nil {
		fmt.Println("Body (Form values):")
		printStrSliceMap(r.BodyFormValues)
		return
	}

	if r.BodyMultipartFormValues != nil {
		fmt.Println("Body (Multipart form values):")
		printStrSliceMap(r.BodyMultipartFormValues.Value)

		for key, values := range r.BodyMultipartFormValues.File {
			fmt.Printf("\t%s:\n", key)
			for _, value := range values {
				fmt.Printf("\t\t%s (%.2f MB)\n", value.Filename, float64(value.Size)/1024/1024)
				if len(value.Header) != 0 {
					fmt.Println("\t\tHeaders:")
					for _, i := range value.Header {
						fmt.Printf("\t\t\t%s\n", i)
					}
				}
			}
		}
		return
	}

	fmt.Printf("Body (unknown): lenght %d bytes\n", len(r.Body))
}

func (r *Request) Print() {
	log.Println(r.Method, r.Path, r.Protocol)

	if r.Path != r.RequestUri {
		fmt.Println("Request URI:", r.RequestUri)
	}

	fmt.Println("Host:", r.Host)
	fmt.Println("Remote Address:", r.RemoteAddress)

	r.printAuth()
	r.printHeaders()
	r.printQueryParams()
	r.printBody()

	const delimiter string = "-------------------------"
	fmt.Println(delimiter)
}

func NewRequest(r *http.Request) (*Request, error) {
	method := r.Method
	if method == "" {
		method = "GET"
	}

	username, password, ok := r.BasicAuth()
	auth := &basicAuth{
		username: username,
		password: password,
		ok:       ok,
	}

	var body []byte
	var bodyParseError error = nil
	var bodyFormValues map[string][]string = nil

	contentType := r.Header.Get("Content-Type")
	bodyIsString := isStringContentType(contentType)

	if isStringContentType(contentType) {
		body, bodyParseError = readBodyAsBytes(r.Body)
	}

	switch contentType {
	case mimetype.ApplicationXWwwFormUrlencoded:
		bodyParseError = r.ParseForm()
		if bodyParseError == nil {
			bodyFormValues = r.Form
		} else {
			bodyParseError = fmt.Errorf("error parsing form values: %v", bodyParseError)
		}

	case mimetype.MultipartFormData:
		bodyParseError = r.ParseMultipartForm(int64(Cfg.MaxFormBodySizeInMB) << 20)
		if bodyParseError != nil {
			bodyParseError = fmt.Errorf("error parsing multipart form values: %v", bodyParseError)
		}
	}

	return &Request{
		Method:                  method,
		Path:                    r.URL.Path,
		RequestUri:              r.RequestURI,
		Protocol:                r.Proto,
		Host:                    r.Host,
		RemoteAddress:           r.RemoteAddr,
		ContentLength:           r.ContentLength,
		ContentType:             contentType,
		Headers:                 r.Header,
		QueryParams:             r.URL.Query(),
		BasicAuth:               auth,
		BodyIsString:            bodyIsString,
		BodyParseError:          bodyParseError,
		Body:                    body,
		BodyFormValues:          bodyFormValues,
		BodyMultipartFormValues: r.MultipartForm,
	}, nil
}

func readBodyAsBytes(body io.ReadCloser) ([]byte, error) {
	b, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %v", err)
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Printf("error closing request body: %v", err)
			return
		}
	}(body)

	return b, nil
}

func isStringContentType(contentType string) bool {
	if strings.HasPrefix(contentType, "text/") {
		return true
	}

	if strings.Contains(contentType, "json") {
		return true
	}

	if strings.Contains(contentType, "xml") {
		return true
	}

	if contentType == mimetype.ApplicationXWwwFormUrlencoded {
		return true
	}

	return false
}

func printStrMap(m map[string]string) {
	for key, value := range m {
		fmt.Printf("\t%s: %s\n", key, value)
	}
}

func printStrSliceMap(m map[string][]string) {
	for key, values := range m {
		for _, value := range values {
			fmt.Printf("\t%s: %s\n", key, value)
		}
	}
}

func printJsonIndented(b []byte) (err error) {
	const prefix string = ""
	const indent string = "  "

	if !json.Valid(b) {
		err = errors.New("invalid json")
		return err
	}

	var jsonStr bytes.Buffer
	err = json.Indent(&jsonStr, b, prefix, indent)
	if err != nil {
		return err
	}

	fmt.Println(jsonStr.String())

	return nil
}

func printBytes(bytes []byte) {
	fmt.Println(string(bytes))
}
