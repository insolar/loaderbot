docker build -t nginx_test .
docker run --name some-nginx --rm -p 8080:80 -v $(pwd)/nginx-cache:/var/cache/nginx nginx_test