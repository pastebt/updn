package main

import (
    "os"
    "io"
    "fmt"
    "path"
    "sort"
    "html"
    "strings"
    "net/url"
    "net/http"
)


func hPost(w http.ResponseWriter, r *http.Request) {
    hUpload(w, r, ".")
    fmt.Fprintf(w, "</body></html>")
}


func hUpload(w http.ResponseWriter, r *http.Request, dir string) {
    fn := ""
    fobj, fh, err := r.FormFile("attachment")
    //_, fh, err := r.FormFile("attachment")
    if err == nil {
        fmt.Println(fh.Filename)
        ns := strings.Split(fh.Filename, `\\`)
        fn = path.Base(ns[len(ns) - 1])
        ln := path.Join(dir, fn)
        fout, err := os.OpenFile(ln, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
        if err != nil {
            fmt.Fprint(w, err)
            return
        }
        defer fout.Close()
        io.Copy(fout, fobj)
    }
    ret := `<html><body>
<form method="post" action="/%s" enctype="multipart/form-data">
Attachment: <input type=file name="attachment"><br>
<input type=submit value="Post"><br>
</form><br>
%s
`
    fmt.Fprintf(w, ret, dir, fn)
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


func dirList(w http.ResponseWriter, r *http.Request, f http.File) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    dirs, err := f.Readdir(-1)
    if err != nil { //|| len(dirs) == 0 {
        return
    }
    hUpload(w, r, path.Join("./", r.RequestURI) + "/")
    fmt.Fprintf(w, "<pre>\n")
    for _, d := range dirs {
        name := d.Name()
        if d.IsDir() {
            name += "/"
        }
        // name may contain '?' or '#', which must be escaped to remain
        // part of the URL path, and not indicate the start of a query
        // string or fragment.
        url := url.URL{Path: name}
        fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n",
                    url.String(), html.EscapeString(name)) //htmlReplacer.Replace(name))
    }
    fmt.Fprintf(w, "</pre>\n</body></html>")
}


type fileHandler struct {
    root http.FileSystem
}


func (fh *fileHandler)ServeHTTP(w http.ResponseWriter, r *http.Request) {
    upath := r.URL.Path
    if !strings.HasPrefix(upath, "/") {
        upath = "/" + upath
        r.URL.Path = upath
    }
    name := path.Clean(upath)
    f, err := fh.root.Open(name)
    if err != nil {
        http.Error(w, "Something Wrong", http.StatusInternalServerError)
        return
    }
    defer f.Close()

    d, err1 := f.Stat()
    if err1 != nil {
        http.Error(w, "Something Wrong", http.StatusInternalServerError)
        return
    }

   if d.IsDir() {
        dirList(w, r, f)
        return
    }
    http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}


var fileList http.Handler


func hUpdn(w http.ResponseWriter, r *http.Request) {
    p := path.Join("./", r.RequestURI)
    hUpload(w, r, p + "/")
    fileList.ServeHTTP(w, r)
    //http.ServeFile(w, r, p)
    fmt.Fprintf(w, "</body></html>")
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
    //fileList = http.FileServer(Root("./"))
    fmt.Printf("serve http at %s\n", os.Args[1])
    //mux.HandleFunc("/", hUpdn)
    fh := fileHandler{Root("./")}
    mux.Handle("/", &fh)
    err := http.ListenAndServe(":" + os.Args[1], mux)
    if err != nil { panic (err) }
}
