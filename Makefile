default: cover

cover:
	go test -v -race -coverprofile cover.out .
	go tool cover -html=cover.out

test:
	go test -v -race .

bench:
	go test -v -benchmem -bench .
