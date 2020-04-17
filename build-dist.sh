#!/usr/bin/env bash

rm -rf ./dist
mkdir ./dist

VER=`git describe --tags --always --long`
echo Version: $VER

platforms=("windows/amd64" "darwin/amd64" "linux/amd64")
for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X 'github.com/qiuyuzhou/h2tun/cmd/h2tun/cmd.version=$VER'" -o ./dist/h2tun-$GOOS-$GOARCH ./cmd/h2tun
done
