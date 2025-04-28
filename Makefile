build: 
	go build -C ./src -o ../bin/kulturtelefon-stream

run: build
	@./bin/kulturtelefon-stream

test:
	cd ./src && go test -v

clean:
	rm -rf ./bin/*
	rm streams.db

init:
	cd ./src && go mod init github.com/anux-linux/kulturtelefon-stream 

tidy:
	cd ./src && go mod tidy