.PHONY : proto proto-python clean build run trace-firecracker trace-container wimpy

proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		pkg/workload/proto/faas.proto
	/usr/bin/python3 -m grpc_tools.protoc -I=. \
		--python_out=. \
		--grpc_python_out=. \
		pkg/workload/proto/faas.proto

proto-python:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		server/trace-func-py/faas.proto
	python3 -m grpc_tools.protoc -I server/trace-func-py/. \
		--python_out=server/trace-func-py/. \
		--grpc_python_out=server/trace-func-py/. \
		faas.proto

# make -i clean
clean: 
# 	kubectl rollout restart deployment activator -n knative-serving

	scripts/util/clean_prometheus.sh

	kn service delete --all
	kubectl delete --all all -n default --grace-period=0 

# 	Deployments should be deleted first!
# 	kubectl delete --all deployments,pods,podautoscalers -n default
# 	kubectl delete --all deployments -n default
# 	kubectl delete --all pods -n default
# 	kubectl delete --all podautoscalers -n default

	bash scripts/warmup/reset_kn_global.sh
	rm -f loader
# 	rm -f *.log
	go mod tidy

rm-results:
	rm data/out/*.csv

build:
	go build cmd/loader.go

run:
	go run cmd/loader.go --config cmd/config.json

test:
	go test -v -cover -race \
		./pkg/config/ \
		./pkg/driver/ \
		./pkg/generator/ \
		./pkg/trace/

# Used for replying the trace
trace-firecracker:
	docker build --build-arg FUNC_TYPE=TRACE \
		--build-arg FUNC_PORT=50051 \
		-f Dockerfile.trace \
		-t cvetkovic/trace_function_firecracker .
	docker push cvetkovic/trace_function_firecracker:latest

# Used for replying the trace
trace-container:
	docker build --build-arg FUNC_TYPE=TRACE \
		--build-arg FUNC_PORT=80 \
		-f Dockerfile.trace \
		-t cvetkovic/trace_function .
	docker push cvetkovic/trace_function:latest

trace-container-py:
	docker build --build-arg FUNC_TYPE=TRACE \
		--build-arg FUNC_PORT=80 \
		-t nehalem90/trace-func-py \
		./server/trace-func-py/.
	docker push nehalem90/trace-func-py:latest

# Used for measuring cold start latency
empty-firecracker:
	docker build --build-arg FUNC_TYPE=EMPTY \
		--build-arg FUNC_PORT=50051 \
		-f Dockerfile.trace \
		-t cvetkovic/empty_function_firecracker .
	docker push cvetkovic/empty_function_firecracker:latest

# Used for measuring cold start latency
empty-container:
	docker build --build-arg FUNC_TYPE=EMPTY \
		--build-arg FUNC_PORT=80 \
		-f Dockerfile.trace \
		-t cvetkovic/empty_function .
	docker push cvetkovic/empty_function:latest

wimpy:
	docker build -f Dockerfile.wimpy -t hyhe/wimpy .
	docker push hyhe/wimpy:latest
