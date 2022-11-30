# GRPC Boilerplate

_Fully functional boilerplate for running a multiplexed GRPC Server and GRPC
Gateway (JSON/REST proxy) over HTTPS, with a Typescript web client._

GRPC is a great way for building web services. Once up and running it offers a
developer experience that is convenient and quick to build on, and with the GRPC
Gateway you can deliver elegant JSON/HTTP APIs to web clients.

However, getting a new server set up can be rather fiddly, and despite numerous
blog posts and walkthroughs it can still take hours of Googling to get it going.

This repo is intended to fast track the set up of new projects.

- https://github.com/grpc/grpc-go
- https://grpc-ecosystem.github.io/grpc-gateway/
- https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_openapi_output/

## What you get

- A `helloworld` GRPC Server based on the [offical example](https://github.com/grpc/grpc-go/tree/master/examples/helloworld).
- A GRPC Gateway configured to proxy the same `helloworld` service.
- A generated Typescript client which talks to the GRPC Gateway, and mimimal
  page that exercises the client.
- A simple multiplexer that allows both the GRPC Gateway and the GRPC Server to
  operate on the same port.
- TLS support for the GRPC Gateway, with self-signed local certs.

## Setup

1. Fork/clone this repo.
2. Ensure protoc is installed `brew install protobuf`.
3. Install dependencies `make install-deps`.
4. Generate SSL certificates `make gen-certs` for development.
5. Follow instructions to add certs to your key chain.
6. Run the server `make run`.
7. Make a test request `curl 'https://localhost:5050/v1/greeting?name=dan'`.
8. Open https://localhost:5050 in your browser, the prompt and alert will make an RPC
9. 🎉🥂

Hopefully that works. Run `make help` to see information about other commands.

Please submit Pull Requests with corrections or updates.

## Code layout

This boilerplate is intended to work best for a multi-language, mono-repo
project. Go source files are rooted under [go/src/boiler.plate/](go/src/boiler.plate/).
I suggest you find-and-replace references to `boiler.plate` with
`myawesomeapp.com` or whatever.

The main entrypoint can be found at [go/src/boiler.plate/cmd/boiler-pate.go](go/src/boiler.plate/cmd/boiler-pate.go).

Proto services defined in [protos/](protos/) will generate interfaces under
[go/src/boiler.plate/services/](go/src/boiler.plate/services/). GRPC allows for
service oriented design, even if you deploy as a monolithic binary. We suggest
you create granular services such as `UserServer` defined in `user.proto` and
`AuthServer` defined in `auth.proto`.

## Contributing

If you find issues or spot possible improvements, please submit a pull-request.
