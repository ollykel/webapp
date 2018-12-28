package webapp

import (
	"fmt"
	"net/http"
	"time"
	"math/rand"
)

const (
	hash_length = 16
	session_cookie_name = "session"
	cookie_format = "%s : %s"
)

type SessionManager struct {
	sessions map[string]string
	randGen *rand.Rand
	sessionDuration int
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
	sm.sessions[key] = hash
	return hash
}//-- end func SessionManager.Login

func (sm *SessionManager) LoginLimited (key string, duration int) string {
	if duration < 1 { panic("duration less than 1 sec") }
	hash := sm.Login(key)
	go func() {
		time.Sleep(time.Duration(duration * 1000))
		sm.Logout(key)
	}()
	return hash
}//-- end func SessionManager.LoginLimited

func (sm *SessionManager) IsLoggedIn(key string) bool {
	_, loggedIn := sm.sessions[key]
	return loggedIn
}//-- end func SessionManager.IsLoggedIn

func (sm *SessionManager) Validate(key, hash string) bool {
	value, exists := sm.sessions[key]
	return exists && value == hash
}//-- end func SessionManager.Validate

func NewSessionManager(sessionDuration int) *SessionManager {
	return &SessionManager{
		randGen: rand.New(rand.NewSource(time.Now().UnixNano())),
		sessions: make(map[string]string),
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
				w.Write([]byte(`{"loginSucceeded": false}`))
				return
			}
			hash := sm.LoginLimited(key, sm.sessionDuration)
			sessionCookie := http.Cookie{
				Name: session_cookie_name,
				Value: fmt.Sprintf(cookie_format, key, hash),
				MaxAge: sm.sessionDuration,
				Secure: true,
				HttpOnly: true}
			w.Write([]byte(`{"loginSucceeded": true}`))
			http.SetCookie(w, &sessionCookie)
		}//-- end return
	}//-- end return
}//-- end func Sessionmanager.LoginHandler

