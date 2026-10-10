IMAGE_REPOSITORY = quay.io/kubermatic-labs
APPLICATION_NAME = training-application
BUILD_VERSION = 4.0.0
BUILD_VERSION_A = ${BUILD_VERSION}-A
BUILD_VERSION_B = ${BUILD_VERSION}-B
BUILD_VERSION_DISTROLESS = ${BUILD_VERSION}-distroless
HELM_CHART_VERSION = 1.0.2

.PHONY: update-dependencies
update-dependencies: 
	cd src && go get -u
	cd src && go mod tidy

.PHONY: lint
lint:
	# cd src && golangci-lint run --timeout=5m -v
	cd src && golangci-lint run --timeout=5m

.PHONY: build
build: 
	cd src && go build -o ../${APPLICATION_NAME} .

.PHONY: run
run: build
	./${APPLICATION_NAME}

.PHONY: docker-lint
docker-lint: 
	hadolint docker/Dockerfile

.PHONY: docker-lint-all
docker-lint-all: 
	hadolint docker/Dockerfile
	hadolint docker/Dockerfile-A
	hadolint docker/Dockerfile-B --ignore DL3025
	hadolint docker/Dockerfile-distroless --ignore DL3006

.PHONY: docker-build
docker-build: build docker-lint
	docker build -f docker/Dockerfile -t ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION} .

.PHONY: docker-run
docker-run: docker-build
	docker run -it --rm -p 8080:8080 -m=10m --cpus=".5" --name ${APPLICATION_NAME} ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION}

.PHONY: docker-build-all
docker-build-all: lint docker-lint
	docker build -f docker/Dockerfile -t ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION} .
	docker build -f docker/Dockerfile-A -t ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION_A} .
	docker build -f docker/Dockerfile-B -t ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION_B} .
	docker build -f docker/Dockerfile-distroless -t ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION_DISTROLESS} .

.PHONY: docker-push
docker-push: 
	docker buildx build --push --platform linux/arm64,linux/amd64  -f docker/Dockerfile --tag ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION} .

.PHONY: docker-push-all
docker-push-all: 
	docker buildx build --push --platform linux/arm64,linux/amd64 -f docker/Dockerfile --tag ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION} .
	docker buildx build --push --platform linux/arm64,linux/amd64 -f docker/Dockerfile-A --tag ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION_A} .
	docker buildx build --push --platform linux/arm64,linux/amd64 -f docker/Dockerfile-B --tag ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION_B} .
	docker buildx build --push --platform linux/arm64,linux/amd64 -f docker/Dockerfile-distroless --tag ${IMAGE_REPOSITORY}/${APPLICATION_NAME}:${BUILD_VERSION_DISTROLESS} .

.PHONY: helm-push
helm-push:
	helm package ./helm-chart --version ${HELM_CHART_VERSION}
	helm push --debug training-application-${HELM_CHART_VERSION}.tgz oci://quay.io/kubermatic-labs/helm-charts/

# run everything locally via kind

.PHONY: kind-start
kind-start:
	# docker-desktop
	kind create cluster --config kind/cluster.yaml
	sudo cloud-provider-kind

.PHONY: run-at-kind
run-at-kind:
	kubectl apply -f kind/gateway.yaml
	kubectl get gateway -A
	helm upgrade --install my-app ./helm-chart --set gateway.enabled=true --set gateway.namespace=gateway --wait 
	kubectl get gateway -A

.PHONY: kind-delete
kind-delete:
	kind delete cluster --name training
