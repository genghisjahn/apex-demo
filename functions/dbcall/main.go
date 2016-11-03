package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"database/sql"

	"github.com/apex/go-apex"
	_ "github.com/go-sql-driver/mysql"
)

var info *log.Logger

func main() {
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		var rEvent Event
		var dbinfo DBInfo
		var m Movie
		var id int
		var idErr error
		xborbits := log.Ldate | log.Ltime | log.Lshortfile
		info = log.New(os.Stderr, "dbLog", xborbits)
		info.Println("Start...MarshalJSON")
		data, err := event.MarshalJSON()
		if err != nil {
			return nil, err
		}
		info.Println("End...MarshalJSON")
		info.Println("Start...Unmarshal")
		jdata := strings.Replace(string(data), `\"`, `"`, -1)
		json.Unmarshal([]byte(jdata), &rEvent)
		info.Println("End...Unmarshal")
		datatype := rEvent.Params.Path["type"]
		if !(datatype == "movie" || datatype == "actor") {
			return nil, fmt.Errorf("Invalid Type %s", datatype)
		}
		rawid := rEvent.Params.Querystring["id"]
		if id, idErr = strconv.Atoi(rawid); idErr != nil {
			return nil, idErr
		}
		dbinfo.Location = rEvent.StageVars["dblocation"]
		dbinfo.DBName = rEvent.StageVars["dbname"]
		dbinfo.Username = rEvent.StageVars["dbuser"]
		dbinfo.Password = rEvent.StageVars["dbpassword"]
		if datatype == "movie" {
			var dbErr error
			info.Println("Start DB Call Movie")
			m, dbErr = getMovieData(id, dbinfo)
			if dbErr != nil {
				info.Println("Error:", dbErr)
				return nil, dbErr
			}
			info.Println("End DB Call Movie")
		}
		info.Println("End")
		return m, nil
	})
}

func getMovieData(id int, dbinfo DBInfo) (Movie, error) {
	constr := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbinfo.Username, dbinfo.Password, dbinfo.Location, dbinfo.DBName)
	m := Movie{}
	db, err := sql.Open("mysql",
		constr)
	defer db.Close()
	if err != nil {
		info.Println("ERROR Open:", err)
		return m, err
	}
	rows, errRow := db.Query("select m.*,c.name as `character`,a.id as actorid,a.lastname,a.firstname from movie m,`character` c,actor a  where c.movieid=m.id and c.actorid=a.id and m.id = ?", id)
	if errRow != nil {
		info.Println("ERROR Row:", errRow)

		return m, errRow
	}
	for rows.Next() {
		var mid int
		var title string
		var year int
		var character string
		var actorid int
		var lastname string
		var firstname string
		if errScan := rows.Scan(&mid, &title, &year, &character, &actorid, &lastname, &firstname); errScan != nil {
			info.Println("ERROR Scan:", errScan)
			return m, errScan
		}
		m.Title = title
		m.Year = year
		c := Character{}
		c.Name = character
		c.ActorInfo = Actor{ID: actorid, LastName: lastname, FirstName: firstname}
		m.Characters = append(m.Characters, c)
	}
	return m, nil
}

type DBInfo struct {
	Location string
	DBName   string
	Username string
	Password string
}

type Movie struct {
	Title      string      `json:"title"`
	Year       int         `json:"year"`
	Characters []Character `json:"characters"`
}

type Character struct {
	Name      string `json:"name"`
	ActorInfo Actor  `json:"info"`
}

type Actor struct {
	ID        int       `json:"actor_id"`
	LastName  string    `json:"last_name"`
	FirstName string    `json:"first_name"`
	DOB       time.Time `json:"dob,omitempty"`
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
