default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# E2E testing with LocalStack
e2e-up:
	docker-compose up -d
	@echo "Waiting for LocalStack to be ready..."
	@bash -c 'for i in {1..30}; do \
		if docker exec fastssm-localstack aws --endpoint-url=http://localhost:4566 --region=us-east-1 ssm describe-parameters --max-results 1 >/dev/null 2>&1; then \
			echo "✓ LocalStack is ready and SSM is functional!"; \
			exit 0; \
		fi; \
		echo "Waiting for SSM service... ($$i/30)"; \
		sleep 2; \
	done; \
	echo "LocalStack/SSM failed to start"; \
	exit 1'

e2e-down:
	docker-compose down -v

e2e-test: e2e-up
	@echo "Running E2E tests..."
	cd tests/e2e && bash run-e2e-tests.sh || (cd ../.. && $(MAKE) e2e-down && exit 1)
	@echo "✓ E2E tests completed successfully!"

e2e-logs:
	docker-compose logs -f localstack

e2e-clean: e2e-down
	rm -rf tmp/localstack
	cd tests/e2e && rm -f *.tfstate* *.log output.json import-test-main.tf update-test-main.tf e2e-test.tfrc

.PHONY: fmt lint test testacc build install generate e2e-up e2e-down e2e-test e2e-logs e2e-clean
