BUILD_CGO_ENABLED  := 0
SERVICE_NAME       := play-with-docker
REBASE_URL         := "github.com/dimaskiddo/play-with-docker"
COMMIT_MSG         := "update improvement"

.PHONY:

.SILENT:

init:
	make clean
	GO111MODULE=on go mod init

init-dist:
	mkdir -p dist

vendor:
	make clean
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod vendor

build:
	make vendor
	CGO_ENABLED=$(BUILD_CGO_ENABLED) go build -ldflags="-s -w" -a -installsuffix nocgo -o $(SERVICE_NAME) .
	echo "Build '$(SERVICE_NAME)' complete."
	cd router/l2
	CGO_ENABLED=$(BUILD_CGO_ENABLED) go build -ldflags="-s -w" -a -installsuffix nocgo -o ../../$(SERVICE_NAME)-router .
	echo "Build '$(SERVICE_NAME)-router' complete."
	cd ../..

run:
	make vendor
	go run .

clean-build:
	rm -f $(SERVICE_NAME)
	rm -f $(SERVICE_NAME)-router

clean:
	make clean-build
	rm -rf vendor

commit:
	make vendor
	make clean
	git add .
	git commit -am $(COMMIT_MSG)

rebase:
	rm -rf .git
	find . -type f -iname "*.go*" -exec sed -i '' -e "s%github.com/dimaskiddo/play-with-docker%$(REBASE_URL)%g" {} \;
	git init
	git remote add origin https://$(REBASE_URL).git

push:
	git push origin master

pull:
	git pull origin master
