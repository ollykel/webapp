package resp

import (
	"net/http"
	"strings"
	"encoding/json"
	"encoding/xml"
)

type Response interface {
	Write (http.ResponseWriter)
}//-- end Response interface

type dataEncoder interface {
	Encode (interface{}) error
}

type dataResponse struct {
	Code int
	Msg string
	Data interface{}
	Cookies []http.Cookie
	prefix string
	encoder dataEncoder
}//-- end dataResponse struct

func (resp *dataResponse) Write (w http.ResponseWriter, content string) {
	if resp.Code == 0 { resp.Code = http.StatusOK }
	if resp.Msg == "" { resp.Msg = http.StatusText(resp.Code) }
	w.WriteHeader(resp.Code)
	w.Header().Set("Content-Type", content)
	if resp.prefix != "" { w.Write([]byte(resp.prefix)) }
	if resp.encoder != nil && resp.Data != nil {
		resp.encoder.Encode(resp.Data)
	}
	if resp.Cookies != nil {
		for _, ck := range resp.Cookies {
			http.SetCookie(w, &ck)
		}//-- end for range resp.Cookies
	}
}//-- end func dataResponse.Write

type JSON dataResponse

func (resp *JSON) Write (w http.ResponseWriter) {
	resp.encoder = json.NewEncoder(w)
	(*dataResponse)(resp).Write(w, "application/json")
}//-- end func JSON.Write

type XML dataResponse

type xmlEncoder xml.Encoder

func (enc *xmlEncoder) Encode (v interface{}) error {
	return (*xml.Encoder)(enc).EncodeElement(v, xml.StartElement{
		Name: xml.Name{Local: "xml"}})
}//-- end func xmlEncoder.Encode

func (resp *XML) Write (w http.ResponseWriter) {
	resp.encoder = (*xmlEncoder)(xml.NewEncoder(w))
	resp.prefix = `<?xml version="1.0" encoding="UTF-8"?>`
	(*dataResponse)(resp).Write(w, "application/xml")
}//-- end func XML.Write

type Data struct {
	Type string
	Code int
	Msg string
	Data interface{}
	Cookies []http.Cookie
}//-- end Data struct

func (d *Data) Resp () Response {
	dataType := strings.ToUpper(d.Type)
	switch (dataType) {
		case "XML":
			return &XML{Code: d.Code, Msg: d.Msg, Data: d.Data,
				Cookies: d.Cookies}
		default:
			return &JSON{Code: d.Code, Msg: d.Msg, Data: d.Data,
				Cookies: d.Cookies}
	}//-- end switch
}//-- end func Data.Resp

func (d *Data) Write (w http.ResponseWriter) {
	response := d.Resp()
	response.Write(w)
}//-- end Data.Write

type Redirect struct {
	Location string
}//-- end Redirect struct

func (rd *Redirect) Write (w http.ResponseWriter) {
	w.WriteHeader(http.StatusSeeOther)
	w.Header().Set("Location", rd.Location)
}//-- end func Redirect.Write

type Text struct {
	Code int
	Content string
}//-- end Text struct

func (txt *Text) Write (w http.ResponseWriter) {
	if txt.Code == 0 { txt.Code = http.StatusOK }
	w.WriteHeader(txt.Code)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(txt.Content))
}//-- end func Text.Write

type HTML struct {
	Code int
	Content []byte
}//-- end HTML struct

func (doc *HTML) Write (w http.ResponseWriter) {
	if doc.Code == 0 { doc.Code = http.StatusOK }
	w.WriteHeader(doc.Code)
	w.Header().Set("Content-Type", "text/html")
	w.Write(doc.Content)
}//-- end func HTML.Write

