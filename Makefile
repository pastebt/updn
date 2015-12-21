#HTML_TPL=$(shell cat updn.tpl)

BIN="updn"
EXE="updn.exe"

all: updn

updn: *.go *.tpl
	@#go build -ldflags '-X "main.html_tpl=$(HTML_TPL)"' -o $(BIN) updn.go
	@go build -o $(BIN) updn.go
	@GOOS=windows GOARCH=amd64 go build -o $(EXE) updn.go

*.go:

*.tpl:

clean:
	@rm -f $(BIN) $(EXE)
