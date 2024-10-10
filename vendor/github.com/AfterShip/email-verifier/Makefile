PKG_FILES=`go list ./... | sed -e 's=github.com/AfterShip/emailverifier/=./='`

CCCOLOR="\033[37;1m"
MAKECOLOR="\033[32;1m"
ENDCOLOR="\033[0m"

.PHONY: all

test:
	@go test -race -covermode atomic -coverprofile=covprofile ./...

detect_race:
	@go test -v -race

lint:
	@printf $(CCCOLOR)"Checking vet...\n"$(ENDCOLOR)
	@go vet .
	@printf $(CCCOLOR)"GolangCI Lint...\n"$(ENDCOLOR)
	@golangci-lint run
