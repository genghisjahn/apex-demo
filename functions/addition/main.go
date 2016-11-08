package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/apex/go-apex"
)

func main() {
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		var s Solution
		var rEvent Event
		xborbits := log.Ldate | log.Ltime | log.Lshortfile
		info := log.New(os.Stderr, "AdditionLog", xborbits)
		data, err := event.MarshalJSON()
		jdata := strings.Replace(string(data), `\"`, `"`, -1)
		json.Unmarshal([]byte(jdata), &rEvent)
		var n1 int
		var n2 int

		var nErr error
		for k, v := range rEvent.Params.Path {
			info.Printf("Path: %s %s\n", k, v)
			if k == "num1" {
				n1, nErr = strconv.Atoi(v)
				if nErr != nil {
					s.Message = fmt.Sprintf("%v", err)
					return s, nErr
				}
			}
			if k == "num2" {
				n2, nErr = strconv.Atoi(v)
				if nErr != nil {
					s.Message = fmt.Sprintf("%v", err)
					return s, nErr
				}
			}
		}
		if err != nil {
			s.Message = fmt.Sprintf("%v", err)
		}
		s.Numbers = append(s.Numbers, n1, n2)
		s.Sum = n1 + n2
		return s, nil
	})
}

//Solution contains info about the solved addition problem
type Solution struct {
	Numbers []int  `json:"numbers"`
	Sum     int    `json:"sum"`
	Message string `json:"message,omitempty"`
}

type Event struct {
	Body_json struct{} `json:"body-json"`
	Context   struct {
		Account_id                      string `json:"account-id"`
		API_id                          string `json:"api-id"`
		API_key                         string `json:"api-key"`
		Authorizer_principal_id         string `json:"authorizer-principal-id"`
		Caller                          string `json:"caller"`
		Cognito_authentication_provider string `json:"cognito-authentication-provider"`
		Cognito_authentication_type     string `json:"cognito-authentication-type"`
		Cognito_identity_id             string `json:"cognito-identity-id"`
		Cognito_identity_pool_id        string `json:"cognito-identity-pool-id"`
		HTTP_method                     string `json:"http-method"`
		Request_id                      string `json:"request-id"`
		Resource_id                     string `json:"resource-id"`
		Resource_path                   string `json:"resource-path"`
		Source_ip                       string `json:"source-ip"`
		Stage                           string `json:"stage"`
		User                            string `json:"user"`
		User_agent                      string `json:"user-agent"`
		User_arn                        string `json:"user-arn"`
	} `json:"context"`
	Params struct {
		Header      map[string]string `json:"header"`
		Path        map[string]string `json:"path"`
		Querystring map[string]string `json:"querystring"`
	} `json:"params"`
	Stage_variables map[string]string `json:"stage-variables"`
}
