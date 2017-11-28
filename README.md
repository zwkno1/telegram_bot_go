## run bot
1. install depend package
./install.sh

2. build
go build *.go

3. run
./bot -c config.json

## for docker
1. build image
docker build --rm -t bot_go .

2. create a configfile at /data/

3. run
docker run -d --net=host --name bot_go -v /data:/data bot_go
