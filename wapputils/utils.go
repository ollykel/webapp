package wapputils

import (
	"log"
	"net/http"
	"strings"
	"encoding/json"
	"html/template"
	"bytes"
)

var (
	file_types = map[string]string{
		"default": "text/plain",
		"txt": "text/plain",
		"html": "text/html",
		"css": "text/css",
		"js": "application/javascript",
		"csv": "text/csv",
		"gif": "image/gif",
		"ico": "image/x-icon",
		"jpeg": "image/jpeg",
		"jpg": "image/jpeg",
		"json": "application/json",
		"mpeg": "video/mpeg",
		"png": "image/png",
		"pdf": "application/pdf",
		"svg": "image/svg+xml",
		"tar": "application/x-tar",
		"tif": "image/tiff",
		"tiff": "image/tiff",
		"wav": "audio/wav",
		"xhtml": "application/xhtml+xml",
		"xml": "application/xml",
		"zip": "application/zip"}//-- end file_types
)

func SetFileType(filename string) string {
	path := strings.Split(filename, ".")
	ext := path[len(path) - 1]
	fileType, exists := file_types[ext]
	log.Printf("File type: %s\n", fileType)
	if !exists { return file_types["default"] }
	return fileType
}//-- end func setFileType

func CacheFileServer (filename string, ctx interface{}) http.HandlerFunc {
	tmp, err := template.ParseFiles(filename)
	if err != nil {
		log.Print(err.Error())
		return http.NotFound
	}
	content := bytes.Buffer{}
	err = tmp.Execute(&content, ctx)
	if err != nil {
		log.Print(err.Error())
		return http.NotFound
	}
	output := content.Bytes()
	fileType := SetFileType(filename)
	return func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("cached server: %s\n", r.URL.Path)
		w.Header().Set("Content-Type", fileType)
		w.Write(output)
	}//-- end return for existing file
}//-- end func cacheFileServer

func ServeJSON(w http.ResponseWriter, r *http.Request, item interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(item)
	if err != nil {
		http.Error(w, "internal server error",
			http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
}//-- end func ServeJSON

