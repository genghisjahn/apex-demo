package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"database/sql"

	"github.com/apex/go-apex"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
)

var info *log.Logger

const movieformat = "m:%v"

func main() {
	apex.HandleFunc(func(event json.RawMessage, ctx *apex.Context) (interface{}, error) {
		var rEvent Event
		var dbinfo DBInfo
		var m Movie
		var id int
		var err error
		xborbits := log.Ldate | log.Ltime | log.Lshortfile
		info = log.New(os.Stderr, "dbLog", xborbits)
		rEvent, err = getEvent(event)
		if err != nil {
			return nil, err
		}

		datatype := rEvent.Params.Path["type"]
		if !(datatype == "movie" || datatype == "actor") {
			return nil, fmt.Errorf("Invalid Type %s", datatype)
		}
		rawid := rEvent.Params.Querystring["id"]
		if id, err = strconv.Atoi(rawid); err != nil {
			return nil, err
		}
		dbinfo, err = getDBInfo(rEvent)
		if err != nil {
			info.Println(err)
			return nil, err
		}
		if datatype == "movie" {
			m, err = getMovie(id, dbinfo)
			if err != nil {
				info.Println("ERROR:", err)
				return nil, err
			}
		}
		return m, nil
	})
}

func getMovie(id int, db DBInfo) (Movie, error) {
	var m Movie
	var err error
	m, err = getMovieRedis(id, db.RedisEndPoint)
	if err != nil {
		return Movie{}, err
	}
	if m.ID != 0 {
		return m, nil
	}
	m, err = getMovieDB(id, db)
	if err != nil {
		return Movie{}, err
	}
	err = saveMovieToRedis(m, db.RedisEndPoint)
	if err != nil {
		return Movie{}, err
	}
	return m, nil
}

func getDBInfo(r Event) (DBInfo, error) {
	var db DBInfo
	db.Location = r.StageVars["dblocation"]
	db.DBName = r.StageVars["dbname"]
	db.Username = r.StageVars["dbuser"]
	db.Password = r.StageVars["dbpassword"]
	db.RedisEndPoint = r.StageVars["redis"]
	if db.Location == "" || db.DBName == "" || db.Username == "" || db.Password == "" || db.RedisEndPoint == "" {
		return DBInfo{}, fmt.Errorf("Invalid DBInfo %v", db)
	}
	return db, nil
}
func getEvent(rawmsg json.RawMessage) (Event, error) {
	var r Event
	var err error
	var data []byte
	data, err = rawmsg.MarshalJSON()
	if err != nil {
		return Event{}, err
	}
	jdata := strings.Replace(string(data), `\"`, `"`, -1)
	err = json.Unmarshal([]byte(jdata), &r)
	if err != nil {
		return Event{}, err
	}
	return r, nil
}

func getMovieRedis(id int, redisendpoint string) (Movie, error) {
	var m Movie
	c, err := redis.Dial("tcp", fmt.Sprintf("%s:6379", redisendpoint))
	if err != nil {
		info.Println("Redis Dial Error:", err, fmt.Sprintf("%s:6379", redisendpoint))
		return m, err
	}
	defer c.Close()
	data, err := redis.String(c.Do("GET", fmt.Sprintf(movieformat, id)))
	if err != redis.ErrNil && err != nil {
		return m, err
	}
	if len(data) == 0 {
		return m, nil
	}
	jerr := json.Unmarshal([]byte(data), &m)
	if jerr != nil {
		return m, jerr
	}
	m.Source = "cache"
	return m, nil
}

func saveMovieToRedis(m Movie, redisendpoint string) error {
	c, err := redis.Dial("tcp", fmt.Sprintf("%s:6379", redisendpoint))
	if err != nil {
		info.Println("Redis Dial Error:", err, fmt.Sprintf("%s:6379", redisendpoint))
		return err
	}
	defer c.Close()
	movieData, jerr := json.Marshal(&m)
	if jerr != nil {
		return jerr
	}
	_, errConn := c.Do("SETEX", fmt.Sprintf(movieformat, m.ID), 10, string(movieData))
	if errConn != nil {
		return errConn
	}
	return nil
}

func getMovieDB(id int, dbinfo DBInfo) (Movie, error) {
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
		m.ID = mid
		m.Title = title
		m.Year = year
		m.Source = "db"
		c := Character{}
		c.Name = character
		c.ActorInfo = Actor{ID: actorid, LastName: lastname, FirstName: firstname}
		m.Characters = append(m.Characters, c)
	}

	return m, nil
}

type DBInfo struct {
	Location      string
	DBName        string
	Username      string
	Password      string
	RedisEndPoint string
}

type Movie struct {
	ID         int         `json:"movie_id"`
	Title      string      `json:"title"`
	Year       int         `json:"year"`
	Characters []Character `json:"characters"`
	Source     string      `json:"source"`
}

type Character struct {
	Name      string `json:"name"`
	ActorInfo Actor  `json:"info"`
}

type Actor struct {
	ID        int    `json:"actor_id"`
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	DOB       string `json:"dob,omitempty"`
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
