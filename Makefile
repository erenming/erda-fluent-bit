all: build

.PHONY: build
build:
	make clean && go build -buildmode=c-shared -o ./out_erda.so .

clean:
	rm -f out_erda.so out_erda.h
