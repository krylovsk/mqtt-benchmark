# How to build this docker

- Modify the Dockerfile
- `docker build . `
- go inside the container and add `bash` (for containernet):
	+ `docker tag <docker_ID> <container_name>`
	+ `docker run -d -it --name pubXX <container_name>`
	+ `docker exec -it pubXX /bin/sh`
	+ `apk add bash`
	+ `docker commit <container_name>`
	+ `docker push <container_name>`
