package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"database/sql"

	"github.com/apex/go-apex"
	_ "github.com/go-sql-driver/mysql"
)

var db DBInfo

func main() {
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		var rEvent Event
		var m Movie
		xborbits := log.Ldate | log.Ltime | log.Lshortfile
		info := log.New(os.Stderr, "dbLog", xborbits)
		data, err := event.MarshalJSON()
		if err != nil {
			return nil, err
		}
		jdata := strings.Replace(string(data), `\"`, `"`, -1)
		json.Unmarshal([]byte(jdata), &rEvent)
		info.Println("movieID: ", rEvent.Params.Querystring["id"])
		info.Println("Path:", rEvent.Params.Path["type"])
		db.Location = rEvent.StageVars["dblocation"]
		db.DBName = rEvent.StageVars["dbname"]
		db.Username = rEvent.StageVars["dbuser"]
		db.Password = rEvent.StageVars["dbpassword"]
		info.Println("Stage Vars:", rEvent.StageVars)
		m.Name = "My Movie"
		m.Year = 2016
		c := Character{}
		c.Name = "Joe Bob"
		c.ActorInfo.DOB = time.Date(1973, time.June, 10, 0, 0, 0, 0, time.UTC)
		c.ActorInfo.LastName = "Wear"
		c.ActorInfo.FirstName = "Jon"
		m.Characters = append(m.Characters, c)
		return m, nil
	})
}

func getMovieData(id int) error {
	constr := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", db.Username, db.Password, db.Location, db.DBName)
	db, err := sql.Open("mysql",
		constr)
	defer db.Close()
	if err != nil {
		return err
	}
	return nil
}

type DBInfo struct {
	Location string
	DBName   string
	Username string
	Password string
}

type Movie struct {
	Name       string      `json:"name"`
	Year       int         `json:"year"`
	Characters []Character `json:"characters"`
}

type Character struct {
	Name      string `json:"name"`
	ActorInfo Actor  `json:"info"`
}

type Actor struct {
	LastName  string    `json:"last_name"`
	FirstName string    `json:"first_name"`
	DOB       time.Time `json:"dob"`
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
	StageVars map[string]string `json:"stage-variables"`
}
