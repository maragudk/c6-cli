.PHONY: build
build: ggml-metal.metal
	CGO_LDFLAGS="-framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders" LIBRARY_PATH=${PWD} C_INCLUDE_PATH=${PWD} go build ./cmd/c6

.PHONY: clean
clean:
	cd go-llama.cpp && make clean
	rm -f ggml-metal.metal

.PHONY: cover
cover:
	go tool cover -html=cover.out

.PHONY: demo
demo:
	vhs demo.tape

go-llama.cpp:
	git clone --recurse-submodules https://github.com/go-skynet/go-llama.cpp

.PHONY: install
install: ggml-metal.metal
	CGO_LDFLAGS="-framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders" LIBRARY_PATH=${PWD} C_INCLUDE_PATH=${PWD} go install ./cmd/c6

ggml-metal.metal: go-llama.cpp
	cd go-llama.cpp && BUILD_TYPE=metal make libbinding.a
	cp go-llama.cpp/build/bin/ggml-metal.metal .
	cp go-llama.cpp/build/bin/ggml-metal.metal ./cmd/c6

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -coverprofile=cover.out -shuffle on ./...

