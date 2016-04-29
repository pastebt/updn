#HTML_TPL=$(shell cat updn.tpl)

BIN="updn"
EXE="updn.exe"
ARM="updn.arm"

all: updn

updn: *.go *.tpl
	@#go build -ldflags '-X "main.html_tpl=$(HTML_TPL)"' -o $(BIN) updn.go
	@echo "linux"
	@go build -o $(BIN) updn.go
	@echo "win"
	@GOOS=windows GOARCH=amd64 go build -o $(EXE) updn.go
	@echo "arm"
	@GOOS=linux GOARCH=arm GOARM=5 go build -o $(ARM) updn.go

*.go:

*.tpl:

clean:
	@rm -f $(BIN) $(EXE) $(ARM)
