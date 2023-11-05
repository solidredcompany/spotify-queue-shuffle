docker build -t spotify-queue-shuffle .
docker save -o spotify-queue-shuffle.tar spotify-queue-shuffle
docker image rm spotify-queue-shuffle

scp spotify-queue-shuffle.tar root@10.150.0.4:

ssh root@10.150.0.4 'docker load -i spotify-queue-shuffle.tar'

rm spotify-queue-shuffle.tar
