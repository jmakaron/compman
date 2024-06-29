.DEFAULT_GOAL := all

.PHONY: all
all: clean compman_go 

.PHONY: compman_go 
compman_go:
	mkdir -p go/compman
	protoc	-I proto									\
			compman/company.proto			\
			--go_out=go --go_opt=paths=source_relative	\
			--go-grpc_out=go --go-grpc_opt=paths=source_relative

.PHONY: clean
clean:
	go clean
	rm -rf go/compman/*.pb.go