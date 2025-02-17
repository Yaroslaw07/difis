build: 
	go build -o bin/difis github.com/Yaroslaw07/difis/cmd/difis

run: build
	./bin/difis

test:
	go test  ./...


