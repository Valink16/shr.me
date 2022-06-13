package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	Warning *log.Logger
	Info    *log.Logger
	Error   *log.Logger
)

const (
	HASH_LENGTH         = 8
	SHORT_URL_LENGTH    = 6
	LONG_URL_MAX_LENGTH = 1024
)

func init() {
	Info = log.New(os.Stdout, "INFO: ", log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

type API struct {
	db       *sql.DB
	sqlStmts map[string]*sql.Stmt
}

func InitAPI(sqlDriverName string, dataSourceName string) (*API, error) {
	if sqlDriverName == "" {
		sqlDriverName = "mysql"
	}

	db, err := sql.Open(sqlDriverName, dataSourceName)
	if err != nil {
		Error.Println("Failed to create connection to SQL database", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		Error.Println("Failed to connect to database", err)
		return nil, err
	}

	sqlStmtsStr := map[string]string{
		"passwordHash_from_username": "select password_hash from users_auth where username = ?",
		"username_from_userId":       "select username from users_auth where userID = ?",
		"userId_from_username":       "select userID from users_auth where username = ?",
		"userData_from_userId":       "select * from users_data where userID = ?",
		"insert_into_users_auth":     "insert into users_auth(username, password_hash) values(?, ?)",
		"insert_into_users_data":     "insert into users_data values(?, ?, ?, ?)",
		"longUrl_from_shortUrl":      "select longURL from links where shortURL = ?",
		"shortUrl_exists":            "select 1 from links where shortURL = ?",
		"username_exists":            "select 1 from users_auth where username = ?",
		"userId_from_shortUrl":       "select userID from links where shortURL = ?",
		"add_to_links":               "insert into links values(?, ?, ?)",
		"links_from_userId":          "select shortURL, longURL from links where userID = ?",
		"delete_from_links":          "delete from links where shortURL = ? and userID = ?",
	}

	sqlStmts := make(map[string]*sql.Stmt)
	api := &API{
		db,
		sqlStmts,
	}

	err = api.AddStatements(sqlStmtsStr)
	if err != nil {
		log.Fatalln("Failed to prepare statements", err)
		return nil, err
	}

	return api, nil
}

// Adds a prepared SQL statement to the map, which will be automatically managed and able to run
func (api *API) AddStatement(name string, query string) error {
	stmt, err := api.db.Prepare(query)
	if err != nil {
		Warning.Printf("Failed to prepare SQL statement: %v, %v\n", query, err)
		return err
	}

	api.sqlStmts[name] = stmt

	return nil
}

func (api *API) AddStatements(stmtsStr map[string]string) error {
	for name, query := range stmtsStr {
		err := api.AddStatement(name, query)
		if err != nil {
			return err
		}
	}

	return nil
}

func (api *API) QueryRow(name string, args []any, dest ...any) error {
	if stmt, exists := api.sqlStmts[name]; exists {
		return stmt.QueryRow(args...).Scan(dest...)
	} else {
		return &NoSuchStatementError{}
	}
}

func (api *API) ExecRow(name string, args ...any) (int64, error) {
	if stmt, exists := api.sqlStmts[name]; exists {
		res, err := stmt.Exec(args...)
		if err != nil {
			Error.Println("Failed to execute statement", err)
			return 0, err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			Warning.Println("Failed to read RowsAffected", err)
		}

		Info.Printf("Affected %d rows\n", affected)
		return affected, nil
	} else {
		return 0, &NoSuchStatementError{}
	}
}

func (api *API) Query(name string, args ...any) (*sql.Rows, error) {
	if stmt, exists := api.sqlStmts[name]; exists {
		rows, err := stmt.Query(args...)
		if err != nil {
			Warning.Println("Failed to query:", err)
			return nil, err
		}

		return rows, nil
	} else {
		return nil, &NoSuchStatementError{}
	}
}

func (api *API) Close() {
	api.db.Close()

	for _, s := range api.sqlStmts { // Closing all statements
		s.Close()
	}
}

// Signs up the user using data in a form, fails if no data in request body
// Returns true if successful
func (api *API) signup(name, age, born, username, password string) error {
	Info.Printf("Attempting to sign up %v", username)

	if name == "" || age == "" || born == "" || username == "" || password == "" {
		log.Println("Invalid input")
		return &InvalidInput{}
	}

	var exists string
	err := api.QueryRow("username_exists", []any{username}, &exists)
	if err != nil && err != sql.ErrNoRows {
		Error.Println("Failed to check if username already exists", err)
		return err
	}

	if exists == "1" {
		Warning.Println("Username already exists", err)
		return &Unauthorized{}
	}

	password_hash := hash([]byte(password), HASH_LENGTH)
	_, err = api.ExecRow("insert_into_users_auth", username, password_hash)
	if err != nil {
		log.Println("Failed to save auth data", err)
		return err
	}

	var newUserId int
	err = api.QueryRow("userId_from_username", []any{username}, &newUserId)
	if err != nil {
		log.Println("Failed to read new userID", err)
		return err
	}

	_, err = api.ExecRow("insert_into_users_data", newUserId, name, age, born)
	if err != nil {
		log.Println("Failed to save user data", err)
		return err
	}

	Info.Printf("Successfully signed up user %v with userID(%d)", username, newUserId)
	return nil
}

// Signs in the user using data in a form, fails if no data in request body
// Returns true if successful
func (api *API) signin(session *Session, username, password string) error {
	Info.Printf("Attempting to login %v", username)
	password_hash := hash([]byte(password), HASH_LENGTH)
	var stored_password_hash []byte

	err := api.QueryRow("passwordHash_from_username", []any{username}, &stored_password_hash)
	if err != nil {
		Error.Println("Failed to get stored password hash", err)
		return &NoSuchUser{}
	}

	log.Println(username, password_hash, stored_password_hash)

	if bytes.Equal(password_hash, stored_password_hash) {
		// w.Write([]byte("Success"))
		var userId int
		err = api.QueryRow("userId_from_username", []any{username}, &userId)
		if err != nil {
			Error.Println("Failed to get userID", err)
			return &NoSuchUser{}
		}

		Info.Printf("SID(%v) associated with user %v with UserID %d. Elevating session access...\n", session.sid, username, userId)
		session.signedIn = true
		session.userId = userId

		return nil
	} else {
		return &Unauthorized{}
	}
}

// Adds a redirect pair into the database
func (api *API) addURL(session *Session, shortUrl string, longUrl string) error {
	if !session.signedIn {
		Info.Printf("Rejecting unauthorized user with SID(%v)\n", session.sid)
		return &Unauthorized{}
	}

	if len(shortUrl) != SHORT_URL_LENGTH {
		Info.Printf("Rejecting unauthorized shortURL length with SID(%v)\n", session.sid)
		return &Unauthorized{}
	}

	if len(longUrl) > LONG_URL_MAX_LENGTH {
		Info.Printf("Rejecting unauthorized longURL length with SID(%v)\n", session.sid)
		return &Unauthorized{}
	}

	var exists string
	err := api.QueryRow("shortUrl_exists", []any{shortUrl}, &exists)
	if err != nil && err != sql.ErrNoRows {
		Error.Println("Failed to read if short link already exists", err)
		return err
	}

	if exists == "1" {
		Info.Println("Rejecting adding existing short URL")
		return &BadRequest{}
	}

	_, err = api.ExecRow("add_to_links", session.userId, shortUrl, longUrl)
	if err != nil {
		Error.Println("Failed to add link pair", err)
		return err
	}
	return nil
}

// Gets all link pairs from a user identified by the session
func (api *API) getURL(session *Session) (res []LinkData, err error) {
	if !session.signedIn {
		return nil, &Unauthorized{}
	}

	rows, err := api.Query("links_from_userId", session.userId)
	if err != nil {
		Error.Println("Failed to get link pair", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var data LinkData
		err = rows.Scan(&data.Short, &data.Long)
		if err != nil {
			break
		}
		res = append(res, data)
	}

	return
}

// Deletes the shortURL and it's associated longURL
func (api *API) deleteURL(session *Session, shortUrl string) error {
	if !session.signedIn {
		Info.Printf("Rejecting unauthorized user with SID(%v)\n", session.sid)
		return &Unauthorized{}
	}

	var userId int
	err := api.QueryRow("userId_from_shortUrl", []any{shortUrl}, &userId)
	if err != nil {
		Error.Printf("Failed to get userID from shortURL(%v), %v", shortUrl, err)
	}

	if userId != session.userId {
		Info.Printf("Rejecting unauthorized user with SID(%v)\n", session.sid)
		return &Unauthorized{}
	}

	affected, err := api.ExecRow("delete_from_links", shortUrl, session.userId)
	if err != nil || affected == 0 {
		Error.Printf("Failed to execute delete_from_links with argument %v, %v\n", shortUrl, err)
		return err
	}
	return nil
}

func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value(SessionKey).(*Session)
	endpoint := strings.Split(r.URL.String(), "?")[0]
	Info.Printf("Processing API request for SID(%v) @ %v, endpoint[%v]", session.sid, r.URL.String(), endpoint)

	switch r.Method {
	case "POST":
		switch endpoint {
		case "auth":
			err := r.ParseForm()
			if err != nil {
				log.Println("Failed to parse form", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			username := r.PostForm.Get("username")
			password := r.PostForm.Get("password")
			err = api.signin(session, username, password)
			if err != nil {
				switch err.(type) {
				case *NoSuchUser, *Unauthorized:
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Username or password incorrect"))
				default:
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal server error"))
				}
				return
			}
			w.WriteHeader(http.StatusOK)

		case "add":
			err := r.ParseForm()
			if err != nil {
				log.Println("Failed to parse form", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			short := r.PostForm.Get("short")
			long := r.PostForm.Get("long")

			Info.Println("Got arguments:", short, long)
			err = api.addURL(session, short, long)
			if err != nil {
				Warning.Println("Got error:", err)
				switch err.(type) {
				case *BadRequest:
					w.WriteHeader(http.StatusBadRequest)
				case *Unauthorized:
					w.WriteHeader(http.StatusUnauthorized)
				default:
					w.WriteHeader(http.StatusInternalServerError)
				}
				w.Write([]byte(err.Error()))
			} else {
				http.Redirect(w, r, "/manage", http.StatusPermanentRedirect)
			}
		case "signup":
			err := r.ParseForm()
			if err != nil {
				Warning.Println("Failed to parse form", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			name := r.PostForm.Get("name")
			age := r.PostForm.Get("age")
			born := r.PostForm.Get("born")
			username := r.PostForm.Get("username")
			password := r.PostForm.Get("password")
			err = api.signup(name, age, born, username, password)
			if err != nil {
				switch err.(type) {
				case *Unauthorized, *InvalidInput:
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Invalid input, please fill all the fields"))
				default:
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal server error"))
				}
				return
			}
			w.WriteHeader(http.StatusOK)
		}
	case "GET":
		switch endpoint {
		case "get":
			res, err := api.getURL(session)
			if err != nil {
				switch err.(type) {
				default:
					w.WriteHeader(http.StatusInternalServerError)
				case *Unauthorized:
					w.WriteHeader(http.StatusUnauthorized)
				}
				return
			}

			resData, err := json.Marshal(res)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.Write(resData)
			}
		default:
			http.Redirect(w, r, "/notfound", http.StatusPermanentRedirect)
		}
	case "DELETE":
		switch endpoint {
		case "delete":
			err := r.ParseForm()
			if err != nil {
				Warning.Println("Failed to parse form", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			short := r.URL.Query().Get("short")
			if short == "" {
				Info.Printf("Empty query")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			Info.Printf("Removing link pair with shortURL: %v\n", short)
			err = api.deleteURL(session, short)
			if err != nil {
				switch err {
				default:
					w.WriteHeader(http.StatusInternalServerError)
				case &Unauthorized{}:
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(fmt.Sprintf("Cannot remove short link %v because you are unauthorized", short)))
				}
				Warning.Printf("Failed to delete %v, %v", short, err)
			}
		}
	}

}
