#!/bin/bash
openssl genrsa -out rootCA.key 2048
openssl req -new -x509 -days 365 -nodes -sha256 -key rootCA.key -subj "/C=CN/ST=GD/L=SZ/O=GRPC Boilerplate/CN=Boilerplate CA" -out rootCA.crt
openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=GD/L=SZ/O=GRPC Boilerplate/CN=localhost" -out server.csr
openssl x509 -req -extfile <(printf "subjectAltName=DNS:localhost,DNS:localhost") -days 365 -in server.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out server.crt
