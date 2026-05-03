cd main
$env:GOOS='windows'; $env:GOARCH='amd64'; go build -o ../build/Hytale-Decompiler-RAG.exe
$env:GOOS='linux'; $env:GOARCH='amd64'; go build -o ../build/Hytale-Decompiler-RAG-linux
cd ..\build