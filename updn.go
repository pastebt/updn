package main

import (
    "os"
    "io"
    "fmt"
    "path"
    "sort"
    "strings"
    "net/http"
)


func hPost(w http.ResponseWriter, r *http.Request) {
    fn := ""
    fobj, fh, err := r.FormFile("attachment")
    //_, fh, err := r.FormFile("attachment")
    if err == nil {
        fmt.Println(fh.Filename)
        ns := strings.Split(fh.Filename, `\\`)
        fn = path.Base(ns[len(ns) - 1])
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


type FI []os.FileInfo


func (f FI)Len() int { return len(f) }
func (f FI)Swap(i, j int) { f[i], f[j] = f[j], f[i] }
func (f FI)Less(i, j int) bool {
    a, b := f[i].IsDir(), f[j].IsDir()
    if a != b { return a }
    return f[i].Name() < f[j].Name()
}


type myFile struct {
    http.File
    dat []os.FileInfo
    cnt int
}


func (f *myFile)Readdir(n int) (fi []os.FileInfo, err error) {
    if f.cnt == 0 {
        f.dat, err = f.File.Readdir(-1)
        if err != nil { return }
        sort.Sort(FI(f.dat))
    }
    l := len(f.dat)
    if f.cnt >= l {
        return nil, io.EOF
    }
    if n > 0 && f.cnt + n < l {
        fi = f.dat[f.cnt:f.cnt + n]
        f.cnt = f.cnt + n
        return
    }
    fi = f.dat[f.cnt:l]
    f.cnt = l
    return
}


type Root string


func (r Root)Open(name string) (http.File, error) {
    f, e := http.Dir(r).Open(name)
    if e != nil { return f, e }
    m := myFile{File: f, cnt: 0}
    return &m, nil
}


func usage() {
    fmt.Printf("Usage: %s http_port\n", os.Args[0])
    os.Exit(1)
}


func main() {
    if len(os.Args) != 2 { usage() }

    mux := http.NewServeMux()
    mux.HandleFunc("/post", hPost)
    mux.Handle("/files/",
               http.StripPrefix("/files/", http.FileServer(Root("./"))))
    //           http.StripPrefix("/files/", http.FileServer(http.Dir("./"))))
    fmt.Printf("serve http at %s\n", os.Args[1])
    err := http.ListenAndServe(":" + os.Args[1], mux)
    if err != nil { panic (err) }
}
