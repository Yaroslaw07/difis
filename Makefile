build: 
	go build -o bin/difis

run: build
	./bin/difis

test:
	go test ./... -v


