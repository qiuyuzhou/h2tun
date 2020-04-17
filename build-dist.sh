#!/usr/bin/env bash

# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

rm -rf ./dist
mkdir ./dist

VER=`git describe --tags --always --long`
echo Version: $VER

sum=""
if hash shasum 2>/dev/null; then
    sum="shasum"
fi


[[ -z $upx ]] && upx="echo pending"
if [[ $upx == "echo pending" ]] && hash upx 2>/dev/null; then
	upx="upx -9"
fi


platforms=("darwin/amd64" "linux/amd64" "linux/386" "windows/amd64" "windows/386")
for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    suffix=""
    if [ "$GOOS" == "windows" ]
    then
        suffix=".exe"
    fi

    env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -ldflags \
        "-X 'github.com/qiuyuzhou/h2tun/cmd/h2tun/cmd.version=$VER'" \
        -o ./dist/h2tun-$GOOS-${GOARCH}${suffix} ./cmd/h2tun
    $upx dist/h2tun-${GOOS}-${GOARCH}${suffix} > /dev/null
    tar -zcf dist/h2tun-${GOOS}-${GOARCH}-$VER.tar.gz dist/h2tun-${GOOS}-${GOARCH}${suffix}
    if [ "$sum" != ""  ]
    then
        cd dist
        $sum h2tun-${GOOS}-${GOARCH}-$VER.tar.gz
        cd ..
    fi
done
