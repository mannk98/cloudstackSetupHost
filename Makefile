# Name of your binary
BINARY_NAME=cloudstackSetupHost

# Default target: build the binary
build:
	CGO_ENABLE=0 go build -o $(BINARY_NAME) .

# Install to $GOPATH/bin (or go env GOPATH fallback)
install:
	go install .

# Clean up binary in local dir
clean:
	rm -f $(BINARY_NAME)

# Run the app
run: build
	./$(BINARY_NAME)
