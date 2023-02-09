DH_USERNAME ?= zix99
DH_PROJECT ?= ha-ad410
COMMIT_SHA := $(shell git rev-parse --short HEAD)

run:
	go run ha-adapters/cmd/ad410

docker-build:
	docker build -t ha-ad410:latest .

docker-push: docker-build
	docker tag ha-ad410:latest ${DH_USERNAME}/${DH_PROJECT}:latest
	docker tag ha-ad410:latest ${DH_USERNAME}/${DH_PROJECT}:git-${COMMIT_SHA}
	docker push ${DH_USERNAME}/${DH_PROJECT}:latest
	docker push ${DH_USERNAME}/${DH_PROJECT}:git-${COMMIT_SHA}

