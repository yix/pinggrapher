.PHONY: run
.SILENT:

server: pinggrapher
	./pinggrapher -server

test: pinggrapher
	ping 192.168.1.1 \
		| awk  '/from/ { split($$7, resArr, "="); print resArr[2]; fflush() }'\
		| ./pinggrapher


run: pinggrapher
	./pinggrapher

pinggrapher: *.go
	go build -i -o pinggrapher
