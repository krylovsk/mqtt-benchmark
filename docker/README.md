# How to build this docker

- Modify the Dockerfile
- `docker build . `
- go inside the container and add `bash` (for containernet):
	+ `docker tag <image_ID> <image_name>`
	+ `docker run -d -it --name pubXX <image_name>`
	+ `docker exec -it pubXX /bin/sh`
	+ `apk add bash`
	+ `docker commit <container_name>`
	+ `docker push <image_name>`

- fast way: 
	+ `docker build -t flipperthedog/go_publisher:file .`
	+ `docker run -it --name pubTest flipperthedog/go_publisher:file apk add bash`
	+ `docker commit pubTest`
	+ `docker push flipperthedog/go_publisher:file`
