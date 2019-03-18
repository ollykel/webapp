package config

type Config struct {
	Port string
	Index string
	StaticDir string
	Database DatabaseConfig//-- see database.go
	Handlers map[string]http.HandlerFunc
}//-- end Config struct



