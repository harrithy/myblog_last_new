# prod

swag init; go build -o myblog_server main.go

$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o myblog_server main.go