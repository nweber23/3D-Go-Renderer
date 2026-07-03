NAME  := scop
GO    := go
BIN   := $(NAME)
MODEL ?= models/42.obj

all: build

build:
	$(GO) build -o $(BIN) ./cmd/scop

run: build
	./$(BIN) $(MODEL)

test:
	$(GO) vet ./...
	$(GO) test -race ./...

bench:
	$(GO) test -bench=. -benchmem ./internal/mathgl/ ./internal/objparser/

clean:
	$(GO) clean

fclean: clean
	rm -f $(BIN)

re: fclean all

.PHONY: all build run test bench clean fclean re
