package main

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type ContextKey int

const (
	SessionKey = ContextKey(0)
)

type Session struct { // Represents the state of a session
	sid      string // Unique session ID, generated with a (May be useless since we already have the sid as a key in the manager)
	userId   int
	signedIn bool
	expiry   time.Time
}

// Creates and returns a session with minimal privileges
func NewLowSession(sid string, lifetime time.Duration) *Session {
	return &Session{
		sid,
		0,
		false,
		time.Now().Add(lifetime),
	}
}

// Session management struct, serves as middleware to track sessions for each http connection
type Manager struct {
	handler    http.Handler
	sessions   map[string]*Session
	mutex      *sync.Mutex // To avoid race conditions on modifying the sessions
	cookieName string
	maxLife    time.Duration // Session lifetime
}

func NewManager(handler http.Handler, cookieName string, maxLife time.Duration, updateDelay time.Duration) (manager *Manager) {
	mutex := new(sync.Mutex)
	sessions := make(map[string]*Session)
	manager = &Manager{
		handler,
		sessions,
		mutex,
		cookieName,
		maxLife,
	}

	go manager.BackgroundUpdate(updateDelay)

	return
}

// Calls Manager.UpdateExpired with delay, run in goroutine
func (manager *Manager) BackgroundUpdate(delay time.Duration) {
	var lastUpdate time.Time
	for {
		lastUpdate = time.Now()
		manager.UpdateExpired()
		sleepDuration := time.Millisecond * time.Duration(delay.Milliseconds()-time.Since(lastUpdate).Milliseconds())
		// Info.Printf("Session updater sleeping for %v seconds\n", sleepDuration.Seconds())
		time.Sleep(sleepDuration)
	}
}

// Removes expired sessions, needs to be called regularly
func (manager *Manager) UpdateExpired() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for sid, session := range manager.sessions {
		if time.Now().After(session.expiry) {
			Info.Printf("SID(%v) has expired, deleting...\n", session.sid)
			delete(manager.sessions, sid)
		}
	}
}

// Creates a new session if not exist and return or returns active session from the provided cookie in the request
func (manager *Manager) GetSession(w http.ResponseWriter, r *http.Request) (session *Session) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" { // No cookie provided, need to create a session id and include it in the cookie
		Info.Println("No SID provided, creating a new session")
		session = manager.createSession(w, r)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session = manager.sessions[sid]
		if session == nil || time.Now().After(session.expiry) { // Provided SID isn't recognized
			Info.Printf("Session with SID(%v) unrecognized, creating new session\n", session)
			session = manager.createSession(w, r)
		}
	}

	return
}

// Creates and returns a Session, sets the SID in the cookie of the response, updates the sessions map
func (manager *Manager) createSession(w http.ResponseWriter, r *http.Request) (session *Session) {
	sid := uuid.NewString()
	Info.Printf("Generating new Session ID for %v: SID(%v)\n", r.RemoteAddr, sid)
	session = NewLowSession(sid, time.Duration(manager.maxLife))
	manager.sessions[sid] = session
	cookie := http.Cookie{Name: manager.cookieName, Value: url.QueryEscape(sid), Expires: session.expiry, Path: "/", SameSite: http.SameSiteStrictMode}

	Info.Printf("Settings cookie for SID(%v)", session.sid)
	http.SetCookie(w, &cookie)
	return
}

func (manager *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Make the session available to the following handlers
	session := manager.GetSession(w, r)
	Info.Printf("Serving SID(%v) @ %v\n", session.sid, r.URL)

	newContext := context.WithValue(r.Context(), SessionKey, session)
	manager.handler.ServeHTTP(w, r.WithContext(newContext))
}

func hash(input []byte, length int) []byte {
	return (argon2.Key(input, nil, 2, 32*1024, 4, uint32(length)))
}
