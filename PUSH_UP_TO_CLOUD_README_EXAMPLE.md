NOTE:
This only works on Wilhelm computer, your mileage might vary

Login:

aws ecr get-login-password --profile armada-prod --region eu-north-1 | docker login --username AWS --password-stdin 593054043164.dkr.ecr.eu-north-1.amazonaws.com

Push and build and docker and up:

docker buildx build --platform linux/amd64 -t 593054043164.dkr.ecr.eu-north-1.amazonaws.com/armadacms:v2 --push .
