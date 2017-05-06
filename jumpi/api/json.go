package api

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"regexp"
)

type JSONRequest struct {
	obj interface{}
}

type JSONResponse struct {
	Status  int         `json:"status"`
	Content interface{} `json:"response"`
}

type JSONResponseError JSONResponse

var (
	LimitRequest int64 = 4096
)

func ParseJsonRequest(r *http.Request, v interface{}) (*JSONRequest, error) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, LimitRequest))
	if err != nil {
		return nil, err
	}

	if err := r.Body.Close(); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return nil, err
	}

	return &JSONRequest{obj: v}, nil
}

func (jr JSONRequest) Validate() error {
	// object needs to be a pointer to a struct
	st := reflect.TypeOf(jr.obj)
	if st.Kind() != reflect.Ptr {
		return errors.New("invalid request")
	}

	st = st.Elem()
	if st.Kind() != reflect.Struct {
		return errors.New("invalid request")
	}

	// iterate over all fields and check if the format is correct, if nothing
	// is specified then we just accept the json decoding
	for i := 0; i < st.NumField(); i += 1 {
		field := st.Field(i)
		format := field.Tag.Get("valid")
		json_name := field.Tag.Get("json")

		if len(json_name) == 0 {
			json_name = field.Name
		}

		if len(format) > 0 && field.Type.Kind() == reflect.String {
			val := reflect.ValueOf(jr.obj).Elem().Field(i).Interface()
			value, ok := val.(string)
			if !ok {
				return errors.New("invalid field \"" + json_name + "\"")
			}

			// check if value complies to format
			if matched, err := regexp.MatchString(format, value); err != nil || !matched {
				return errors.New("invalid field \"" + json_name + "\"")
			}
		}
	}

	return nil
}

func (jr JSONResponse) Write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(jr.Status)

	if err := json.NewEncoder(w).Encode(jr); err != nil {
		return err
	}

	return nil
}

func ResponseError(w http.ResponseWriter, status int, e error) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(e.Error()); err != nil {
		log.Fatalf("api_server: unable to send error response: %s\n", err.Error())
	}
}
