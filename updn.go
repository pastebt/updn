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
    if err == nil {
        fmt.Println(fh.Filename)
        // win7 upload filename with whole path, cut dir part
        ns := strings.Split(fh.Filename, `\\`)
        fn = path.Base(ns[len(ns) - 1])
        // win7 file name can not have these chars
        // replace them with '_'
        fs := strings.FieldsFunc(fn, func (c rune) bool {
              return strings.ContainsRune(`\\/:*?"<>|`, c)})
        fn = strings.Join(fs, "_")
        ln := path.Join(dir, fn)
        //println("ln =", ln)
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


func showSize(i int64) (s string) {
    if i < 10000 {
        s = fmt.Sprintf("%d", i)
        return
    }
    k := i / 1000
    //l = fmt.Sprintf("%d", k)
    ss := make([]string, 0, 3)
    if k > 1000000 {
        ss = append(ss, fmt.Sprintf("%d", k / 1000000))
        k = k % 1000000
    }
    if k > 1000 {
        ss = append(ss, fmt.Sprintf("%d", k / 1000))
        k = k % 1000
    }
    ss = append(ss, fmt.Sprintf("%dKB", k))
    s = strings.Join(ss, ",")
    return
}


func dirList(w http.ResponseWriter, r *http.Request, f http.File) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    dirs, err := f.Readdir(-1)
    if err != nil && err != io.EOF { //|| len(dirs) == 0 {
        fmt.Printf("dirs=%v, err=%v\n", dirs, err)
        return
    }
    //println("r.RequestURI = ", r.RequestURI)
    rr, err := url.QueryUnescape(r.RequestURI)
    if err != nil {
        fmt.Printf("QueryUnescape, %v, err = %v", r.RequestURI, err)
        return
    }
    //println("rr = ", rr)
    hUpload(w, r, path.Join("./", rr) + "/")
    fmt.Fprintf(w, `<table>
    <thead><tr><th>Name</th><th>Last modified</th><th>Size</th></tr><thead>
    <tbody>`)
    for _, d := range dirs {
        fmt.Fprintf(w, "<tr>\n")
        name := d.Name()
        if d.IsDir() {
            name += "/"
        }
        // name may contain '?' or '#', which must be escaped to remain
        // part of the URL path, and not indicate the start of a query
        // string or fragment.
        url := url.URL{Path: name}
        fmt.Fprintf(w, "<td><a href=\"%s\">%s</a></td>\n",
                    url.String(), html.EscapeString(name)) //htmlReplacer.Replace(name))
        fmt.Fprintf(w, "<td style='padding-left:1em;'>%s</td>\n",
                    d.ModTime().Format("2006-01-02 15:04:05"))
        fmt.Fprintf(w, "<td align='right' style='padding-left:1em;'>%s</td>\n",
                    showSize(d.Size()))
        fmt.Fprintf(w, "</tr>\n")
    }
    fmt.Fprintf(w, "</tbody></table>\n</body></html>")
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


/*
var fileList http.Handler


func hUpdn(w http.ResponseWriter, r *http.Request) {
    p := path.Join("./", r.RequestURI)
    hUpload(w, r, p + "/")
    fileList.ServeHTTP(w, r)
    //http.ServeFile(w, r, p)
    fmt.Fprintf(w, "</body></html>")
}
*/

//var html_tpl string


func usage() {
    //fmt.Printf("%s\n", html_tpl)
    fmt.Printf("Usage: %s http_port [template_file]\n", os.Args[0])
    os.Exit(1)
}


func main() {
    if len(os.Args) != 2 && len(os.Args) != 3 { usage() }

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
