package main

import (
    "os"
    "io"
    "fmt"
    "path"
    "strings"
    "net/http"
)


func hPost(w http.ResponseWriter, r *http.Request) {
    fn := ""
    fobj, fh, err := r.FormFile("attachment")
    //_, fh, err := r.FormFile("attachment")
    if err == nil {
        //fn = path.Base(fh.Filename)
        fmt.Println(fh.Filename)
        ns := strings.Split(fh.Filename, `\\`)
        fn = path.Base(ns[len(ns) - 1])
        //ln := path.Join(".", "atta", fn)
        ln := path.Join(".", fn)
        fout, err := os.OpenFile(ln, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
        if err != nil {
            fmt.Fprint(w, err)
            return
        }
        defer fout.Close()
        io.Copy(fout, fobj)
    }
    ret := `<html><body>
<form method="post" action="/post" enctype="multipart/form-data">
Attachment: <input type=file name="attachment"><br>
<input type=submit value="Post"><br>
</form><br>
%s
</body>
</html>`
    fmt.Fprintf(w, ret, fn)
}


func usage() {
    fmt.Printf("Usage: %s http_port\n", os.Args[0])
    os.Exit(1)
}


func main() {
    if len(os.Args) != 2 { usage() }

    mux := http.NewServeMux()
    mux.HandleFunc("/post", hPost)
    mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir("./"))))
    fmt.Printf("serve http at %s\n", os.Args[1])
    err := http.ListenAndServe(":" + os.Args[1], mux)
    if err != nil { panic (err) }
}
