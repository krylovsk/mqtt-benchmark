version=$1 
docker build --no-cache --tag flipperthedog/go_publisher:$version .
container_id=$(docker run -it --rm --detach flipperthedog/go_publisher:$version)
docker exec -it $container_id apk add bash
docker commit $container_id
docker push flipperthedog/go_publisher:$version
docker rm -f $container_id
