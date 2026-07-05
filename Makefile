docker-build:
	docker build -t postigo:latest .

docker-run: docker-build
	docker run --rm -ti -p 9090:9090 postigo:latest