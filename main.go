package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	SESSION_MANAGER_UPDATE_DELAY = 30 * time.Minute
)

func main() {
	htmlBase, err := loadTemplateFile("./html/base.template.html")
	if err != nil {
		log.Println("Failed to load template", err)
	}

	managePageBase, err := loadTemplateFile("./html/account/manage.template.html")
	if err != nil {
		log.Println("Failed to load template", err)
	}

	if err != nil {
		Error.Fatalln("Failed to parse template", err)
	}

	api, err := InitAPI("", "root:W.dVTc_+;7JC@tcp(localhost:3306)/test")
	if err != nil {
		log.Fatalln("Failed to create init API", err)
	}
	defer api.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			shortUrl := r.URL.Path[1:]
			var longUrl string
			err = api.QueryRow("longUrl_from_shortUrl", []any{shortUrl}, &longUrl)
			if err != nil {
				Warning.Println("Failed to query longUrl", err)
				http.Redirect(w, r, "/notfound", http.StatusPermanentRedirect)
			}

			Info.Printf("Received request for short link %v, redirecting to %v\n", shortUrl, longUrl)
			http.Redirect(w, r, longUrl, http.StatusPermanentRedirect)
		} else {
			htmlBase.WriteFile("./html/index.html", w)
		}
	})

	mux.Handle("/api/", http.StripPrefix("/api/", api))

	mux.HandleFunc("/notfound", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		htmlBase.WriteFile("./html/notfound.html", w)
	})

	mux.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusPermanentRedirect)
	})

	mux.HandleFunc("/manage", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Check if logged in, if not redirect to login page
		session := r.Context().Value(SessionKey).(*Session)
		if !session.signedIn {
			http.Redirect(w, r, "/signin?redirect=/manage", http.StatusTemporaryRedirect)
			return
		}

		var userData UserData
		err = api.QueryRow("userData_from_userId", []any{session.userId}, &userData.Id, &userData.Name, &userData.Age, &userData.Born)
		if err != nil {
			Error.Println("Failed to get user data", err)
			http.Redirect(w, r, "/notfound", http.StatusPermanentRedirect)
			return
		}

		data, err := api.getURL(session)
		if err != nil {
			Error.Println("Failed to get link data", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		managePageData := &ManagePageData{userData, data}
		managePageOutput, err := managePageBase.ApplyToData(managePageData)
		if err != nil {
			Error.Println("Failed to apply template", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		htmlBase.WriteData(template.HTML(managePageOutput), w)
	})

	mux.HandleFunc("/signin", func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(SessionKey).(*Session).signedIn {
			http.Redirect(w, r, "/", http.StatusPermanentRedirect)
			return
		}
		htmlBase.WriteFile("./html/account/login.html", w)
	})

	mux.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		htmlBase.WriteFile("./html/account/signup.html", w)
	})

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "./static/transparent.ico") })

	static_server := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/", http.StripPrefix("/static/", static_server)) // Removes the /static/ from the file names so the file server gets the correct names

	sessionManager := NewManager(mux, "session_id", time.Hour, SESSION_MANAGER_UPDATE_DELAY)

	Info.Println("Listening...")
	err = http.ListenAndServeTLS(":443", "server.crt", "private.key", sessionManager)
	if err != nil {
		Error.Fatalln("ListenAndServe: ", err)
	}
}
