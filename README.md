# HTTP/2 tunnel - A plugin for shadowsocks

## Intro

[h2tun](https://github.com/qiuyuzhou/h2tun) is a tunnel program over http/2 protocol,
which has been developed as a plugin for shadowsocks. Its approach is similar to trojan,
but it is just a tcp tunnel over http/2, there is no proxy feature.

## Features

* Tunnel over http/2 with TLS
* Tunnel over http/2 cleartext

## Usage

The possible options can pass to the env var `SS_PLUGIN_OPTIONS`.

Shared Options:

| Name | Description |
| --- | --- |
| path={value} | Specify a handle path for tunnel. Default is `/h2tunnel`. You should always specify a secret value. |

Client Mode Options:

| Name | Description |
| --- | --- |
| tls | Connect server by http/2 with TLS. Otherwise over http/2 cleartext |

Server Mode Options:

| Name | Description |
| --- | --- |
| server | Run in serve mode. Otherwise run in client mode.|
| keyFile={value} | The tls cert key file path. |
| certFile={value} | The tls cert file path. |

## Deployment

Recommend to deply a `http reverse proxy` in front of the server.
The `http reverse proxy` should support proxy to upsream by `h2c`.

* [Apache Tttp Server](https://httpd.apache.org/) with mod_http2
* [Traefik](https://docs.traefik.io/)

`h2tun <--[h2]--> http reverse proxy <--[h2c]--> h2tun`

* `h2` is HTTP/2 over TLS (protocol negotiation via ALPN).
* `h2c` is HTTP/2 over TCP.
