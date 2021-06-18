profile:
	go build
	./searchRace
	go tool pprof -http=localhost:9876 searchRace out.prof