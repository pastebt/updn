package main

import (
    "os"
    "io"
    "io/ioutil"
    "fmt"
    "path"
    "sort"
    "html"
    "strings"
    "net/url"
    "net/http"
    //"path/filepath"
)


/*
func hPost(w http.ResponseWriter, r *http.Request) {
    hUpload(w, r, ".")
    fmt.Fprintf(w, "</body></html>")
}
*/


func normWinName(src string) string {
    // win7 file name can not have these chars
    // replace them with '_'
    fs := strings.FieldsFunc(src, func (c rune) bool {
          return strings.ContainsRune(`\\/:*?"<>|`, c)})
    return strings.Join(fs, "_")
}


func hUpload(w http.ResponseWriter, r *http.Request, dir string) (msg string) {
    ct := r.Header.Get("Content-Type")
    if r.Method != "POST" || !strings.HasPrefix(ct, "multipart/") { return }
    mr, err := r.MultipartReader()
    if err != nil {
        msg = "MultipartReader: " + err.Error()
        return
    }
    for {
        p, err := mr.NextPart()
        if err == io.EOF { break }
        if err != nil {
            msg = "NextPart: " + err.Error()
            return
        }
        name := p.FormName()
        switch name {
        case "newfolder":
            bn, err := ioutil.ReadAll(p)
            if err != nil {
                msg = "newfolder ReadAll: " + err.Error()
                return
            }
            nf := strings.TrimSpace(string(bn))
            if len(nf) == 0 { continue }
            nf = normWinName(nf)
            dn := path.Join(dir, nf)
            if err := os.Mkdir(dn, 0700); err != nil {
                //fmt.Fprint(w, err)
                msg = "newfolder Mkdir: " + err.Error()
                return
            }
            msg = "new folder " + nf + "<br>"
            return
        case "attachment":
            fn := strings.TrimSpace(p.FileName())
            if len(fn) == 0 { continue }
            ns := strings.Split(fn, `\\`)
            fn = path.Join(dir, normWinName(path.Base(ns[len(ns) - 1])))
            fout, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
            if err != nil {
                msg = "attachment OpenFile: " + err.Error()
                return
            }
            defer fout.Close()
            io.Copy(fout, p)
            msg = "upload file " + fn + "<br>"
            return
        }
    }
    return
}
//    fobj, fh, err := r.FormFile("attachment")
//    if err == nil {
//        defer fobj.Close()
//        fmt.Println(fh.Filename)
//        // win7 upload filename with whole path, cut dir part
//        ns := strings.Split(fh.Filename, `\\`)
//        fn := path.Base(ns[len(ns) - 1])
//        fn = normWinName(fn)
//        ln := path.Join(dir, fn)
//        //println("ln =", ln)
//        fout, err := os.OpenFile(ln, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
//        if err != nil {
//            fmt.Fprint(w, err)
//            return
//        }
//        defer fout.Close()
//        io.Copy(fout, fobj)
//        msg = "upload file " + fn + "<br>"
//    } else {
//        nf := strings.TrimSpace(r.FormValue("newfolder"))
//        if nf != "" {
//            nf = normWinName(nf)
//            dn := path.Join(dir, nf)
//            if err := os.Mkdir(dn, 0700); err != nil {
//                fmt.Fprint(w, err)
//                return
//            }
//            msg = "new folder " + nf + "<br>"
//        }
//    }
//}


func hUploadPage(w http.ResponseWriter, r *http.Request, dir string) {
    msg := hUpload(w, r, dir)
    ret := `<html><body>
<form method="post" action="/%s" enctype="multipart/form-data">
Attachment: <input type=file name="attachment"><br>
New Folder: <input type=input name="newfolder"><br>
<input type=submit value="Post"><br>
</form><br>
%s
`
    fmt.Fprintf(w, ret, dir, msg)
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
    ss := make([]string, 0, 3)
    if i < 102400 { 
        s = fmt.Sprintf("%0.1f", float64(i) / 1024.0)
    } else {
        s = fmt.Sprintf("%d", (i + 512) / 1024)
        if len(s) > 6 { ss, s = append(ss, s[:len(s) - 6]), s[len(s) - 6:] }
        if len(s) > 3 { ss, s = append(ss, s[:len(s) - 3]), s[len(s) - 3:] }
    }
    ss = append(ss, s + " KB")
    s = strings.Join(ss, ",")
    return
}


func dirList(w http.ResponseWriter, r *http.Request, f http.File, ddot os.FileInfo) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")

    rr, err := url.QueryUnescape(r.RequestURI)
    if err != nil {
        fmt.Printf("QueryUnescape, %v, err = %v", r.RequestURI, err)
        return
    }
    // handle command/upload
    hUploadPage(w, r, path.Join("./", rr) + "/")
    // list files
    dirs, err := f.Readdir(-1)
    if err != nil && err != io.EOF { //|| len(dirs) == 0 {
        fmt.Printf("dirs=%v, err=%v\n", dirs, err)
        return
    }
    fmt.Fprintf(w, `<table>
    <thead><tr><th>Name</th><th>Last modified</th><th>Size</th></tr><thead>
    <tbody>`)
    if ddot != nil {
        fmt.Fprintf(w, "<tr>\n<td><a href=\"../\">..</a></td>\n")
        fmt.Fprintf(w, "<td/><td/>\n</tr>\n")
    }
    for _, d := range dirs {
        fmt.Fprintf(w, "<tr>\n")
        name := d.Name()
        if d.IsDir() {
            name += "/"
        }
        // name may contain '?' or '#', which must be escaped to remain
        // part of the URL path, and not indicate the start of a query
        // string or fragment.
        us := url.URL{Path: name}
        fmt.Fprintf(w, "<td><a href=\"%s\">%s</a></td>\n",
                    us.String(), html.EscapeString(name))
        fmt.Fprintf(w, "<td style='padding-left:1em;'>%s</td>\n",
                    d.ModTime().Format("2006-01-02 15:04:05"))
        fmt.Fprintf(w, "<td align='right' title='%d' " +
                       "style='padding-left:1em;'>%s</td>\n",
                    d.Size(), showSize(d.Size()))
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
    var ddot os.FileInfo
    if len(name) > 1 {
        if fdot, err := fh.root.Open(name + "/.."); err == nil {
            ddot, _ = fdot.Stat()
        }
    }
    defer f.Close()

    d, err1 := f.Stat()
    if err1 != nil {
        http.Error(w, "Something Wrong", http.StatusInternalServerError)
        return
    }

   if d.IsDir() {
        dirList(w, r, f, ddot)
        return
    }
    http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}


func usage() {
    //fmt.Printf("%s\n", html_tpl)
    fmt.Printf("Usage: %s http_port [template_file]\n", os.Args[0])
    os.Exit(1)
}


func main() {
    if len(os.Args) != 2 && len(os.Args) != 3 { usage() }

    mux := http.NewServeMux()
    fmt.Printf("serve http at %s\n", os.Args[1])
    fh := fileHandler{Root("./")}
    mux.Handle("/", &fh)
    err := http.ListenAndServe(":" + os.Args[1], mux)
    if err != nil { panic (err) }
}
