package webapp

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"math/rand"
)

const (
	hash_length = 16
	session_cookie_name = "session"
	cookie_format = "%s : %s"
)

type sessionData struct {
	hash string
	logoutTime int64
}//-- end sessionData struct

func (sd *sessionData) Hash() string { return sd.hash }

func (sd *sessionData) ShouldLogout(tm time.Time) bool {
	return sd.logoutTime < tm.UnixNano()
}//-- end func sessionData.ShouldLogout

type SessionManager struct {
	sessions map[string]sessionData
	toExclude []string
	randGen *rand.Rand
	sessionDuration int//-- in seconds
}//-- end SessionManager struct

func randString(r *rand.Rand, length int) string {
	if length < 1 { return "" }
	output := make([]byte, length)
	for i := range output {
		output[i] = byte(r.Intn(int('Z' - 'A')) + int('A'))
	}//-- end for range output
	return string(output)
}//-- end func randString

func (sm *SessionManager) Logout(key string) {
	delete(sm.sessions, key)
}//-- end func SessionManager.Logut

func (sm *SessionManager) Login(key string) string {
	sm.randGen.Seed(time.Now().UnixNano())
	hash := randString(sm.randGen, hash_length)
	sm.sessions[key] = sessionData{
		hash: hash,
		logoutTime: time.Now().UnixNano() +
			int64(sm.sessionDuration * int(time.Millisecond))}
	return hash
}//-- end func SessionManager.Login

func (sm *SessionManager) LoginLimited (key string, duration int) string {
	if duration < 1 { panic("duration less than 1 sec") }
	hash := sm.Login(key)
	go func() {
		var (
			data sessionData
			exists bool
		)
		for true {
			time.Sleep(time.Duration(duration * int(time.Millisecond)))
			data, exists = sm.sessions[key]
			if !exists || data.ShouldLogout(time.Now()) {
				sm.Logout(key)
				break
			}
		}//-- end for true
	}()
	return hash
}//-- end func SessionManager.LoginLimited

func (sm *SessionManager) IsLoggedIn(key string) bool {
	_, loggedIn := sm.sessions[key]
	return loggedIn
}//-- end func SessionManager.IsLoggedIn

func (sm *SessionManager) CheckRequestLogin (r *http.Request) bool {
	sessionCookie, err := r.Cookie(session_cookie_name)
	if err != nil { return false }
	var key, hash string
	fmt.Sscanf(sessionCookie.Value, cookie_format, &key, &hash)
	return sm.IsLoggedIn(key)
}//-- end SessionManager.CheckRequestLogin

func (sm *SessionManager) Validate(key, hash string) bool {
	value, exists := sm.sessions[key]
	return exists && value.Hash() == hash
}//-- end func SessionManager.Validate

func NewSessionManager(sessionDuration int,
		excluded []string) *SessionManager {
	return &SessionManager{
		randGen: rand.New(rand.NewSource(time.Now().UnixNano())),
		toExclude: excluded,
		sessions: make(map[string]sessionData),
		sessionDuration: sessionDuration}//-- end return
}//-- end func NewSessionManager

func handleShouldLogin(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"shouldLogin": true}`))
}//-- end handleShouldLogin 

func (sm *SessionManager) Middleware (app *Webapp) Middleware {
	manager := sm
	return func(w http.ResponseWriter, r *http.Request) bool {
		for _, path := range sm.toExclude {
			if (len(r.URL.Path) >= len(path) &&
				r.URL.Path[:len(path)] == path) { return true }
		}//-- check if url excluded
		sessionCookie, err := r.Cookie(session_cookie_name)
		if err != nil {
			handleShouldLogin(w, r)
			return false
		}
		var name, hash string
		_, err = fmt.Sscanf(sessionCookie.Value, cookie_format,
			&name, &hash)
		if err != nil {
			handleShouldLogin(w, r)
			return false
		}
		if manager.Validate(name, hash) { return true }
		handleShouldLogin(w, r)
		return false
	}//-- end return
}//-- end func SessionManager.Middleware

type LoginValidator func(*http.Request) (bool, string)

func (sm *SessionManager) LoginHandler (validator func(
		app *Webapp) LoginValidator) AppHandler {
	return func(app *Webapp) http.HandlerFunc {
		validateLogin := validator(app)
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			isValid, key := validateLogin(r)
			if !isValid {
				log.Print("Failed login")
				w.Write([]byte(`{"loginSuccess": false}`))
				return
			}
			log.Print("Successful login")
			hash := sm.LoginLimited(key, sm.sessionDuration)
			sessionCookie := http.Cookie{
				Name: session_cookie_name,
				Value: fmt.Sprintf(cookie_format, key, hash),
				MaxAge: sm.sessionDuration,
				Secure: true,
				HttpOnly: true}
			w.Write([]byte(`{"loginSuccess": true}`))
			http.SetCookie(w, &sessionCookie)
		}//-- end return
	}//-- end return
}//-- end func Sessionmanager.LoginHandler

