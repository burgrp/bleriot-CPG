.PHONY: build flash gdb rtt install-pack disassembly new hub

build:
	go run ./cmd/dev node build --name cpg

flash:
	go run ./cmd/dev node build --name cpg --flash --rtt

disassembly:
	go run ./cmd/dev node build --name cpg --disassembly

gdb rtt install-pack:
	go run ./cmd/dev node $@ --name cpg

new:
	go run ./cmd/dev node new

hub:
	go run ./cmd/dev hub --diagnostics rf