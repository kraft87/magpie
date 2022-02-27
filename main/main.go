package main


import (
	"fmt"
	"io"
	fs "main/fs"
	"net/http"
	"os"
	auth "main/auth"
)

const FILE_MARKER =
`
`
const VERSION = "MagpieFS v1.2"

type wrapperW struct {
	http.ResponseWriter
}

func (w *wrapperW) ReadFrom(src io.Reader) (int64, error) {
	// if its a file, add the file marker length to its size
	if lr, ok := src.(*io.LimitedReader); ok {
		lr.N += int64(len(FILE_MARKER))
	}

	if w, ok := w.ResponseWriter.(interface{ ReadFrom(src io.Reader) (int64, error) }); ok {
		return w.ReadFrom(src)
	}

	panic("unreachable")
}

func main() {
	var path string
	if len(os.Args) < 2 {
		fmt.Println("Defaulting to serving current directory. Use ./fs <path> to serve different")
		path, _ = os.Getwd()
	} else {
		path = os.Args[1]
	}

	fileServ := http.FileServer(fs.CreateFileServFS(
		path,
		// Modify all responses by adding the file marker to them
		// we also need to adjust the file size in the ReadFrom because of to that...
		func(in []byte) (out []byte) {
			out = append(in, FILE_MARKER...)
			return
		}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Served-by", VERSION)
		w = &wrapperW{w}
		fileServ.ServeHTTP(w, r)
	})

	http.HandleFunc("/a_secret.bak", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Served-by", VERSION)
		w.Write([]byte(`Error: Access to secret key only for authorized users.`))
	})

	http.HandleFunc("/flag", func(w http.ResponseWriter, r *http.Request) {
		tokenLine := ""
		w.Header().Set("Served-by", VERSION)
		authHeader, ok := r.Header["Authorization"] // Requires auth
		if ok {
			username, err := auth.TrimAndParseToken([]byte(authHeader[0]))
			if err != nil {
				tokenLine = err.Error()
			} else {
				tokenLine = username
			}
		} else {
			tokenLine = "No token provided"
		}

		if tokenLine == "admin" { // Require the admin user
			fileServ.ServeHTTP(w, r)
		} else {
			w.Write([]byte(`Error: Access to the flag only for the user admin.  `))
			w.Write([]byte("("))
			w.Write([]byte(tokenLine))
			w.Write([]byte(")"))
		}
	})

    port := "8080"
    fmt.Println("Hosting on port", port)
	err := http.ListenAndServe(":"+port, nil)
	fmt.Println(err)
}
