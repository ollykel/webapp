package mux

import (
	"strings"
	"net/http"
	"../resp"
)

type ReqData map[string][]byte

type HandlerFunc func (*http.Request, ReqData) resp.Response

type wildcard struct {
	Name string
	Dest *node
}//-- end wildcard struct

func isWildcard (path string) bool {
	if len(path) < 2 { return false }
	return path[0] == ':'
}//-- end func isWildcard

type node struct {
	Handler HandlerFunc
	Branches map[string]*node
	Wild *wildcard
}//-- end node struct

func (nd *node) Handle (paths []string, handler HandlerFunc) {
	if len(paths) == 0 {
		nd.Handler = handler
		return
	}
	nxt := &node{Branches: make(map[string]*node)}
	nxt.Handle(paths[1:], handler)
	name := paths[0]
	if isWildcard(name) {
		nd.Wild = &wildcard{Name: name[1:], Dest: nxt}
		return
	}
	nd.Branches[name] = nxt
}//-- end func node.Handle

func notFound (_ *http.Request, _ ReqData) resp.Response {
	return &resp.Text{Code: http.StatusNotFound, Content: "not found"}
}//-- end func notFound

func (nd *node) Retrieve (paths []string) HandlerFunc {
	if len(paths) == 0 {
		if nd.Handler == nil { return notFound }
		return nd.Handler
	}
	name := paths[0]
	curr := nd.Branches[name]
	if curr == nil {
		if curr.Wild == nil {
			return notFound
		} else { curr = nd.Wild.Dest }
	}
	return curr.Retrieve(paths[1:])
}//-- end func node.Retrieve

type Mux struct {
	root *node
}//-- end Mux struct

func (mux *Mux) ServeHTTP (w http.ResponseWriter, r *http.Request) {
	if mux.root == nil {
		notFound(r, nil).Write(w)
		return
	}
	if len(r.URL.Path) < 2 {
		mux.root.Handler(r, make(ReqData)).Write(w)
		return
	}
	paths := strings.Split(r.URL.Path[1:], "/")
	handler := mux.root.Retrieve(paths)
	if handler != nil { handler(r, make(ReqData)).Write(w) }
}//-- end Mux.ServeHTTP

