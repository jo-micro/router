[![Build Status](https://drone.fk.jochum.dev/api/badges/jo-micro/router/status.svg)](https://drone.fk.jochum.dev/jo-micro/router) [![Go Reference](https://pkg.go.dev/badge/jochum.dev/jo-micro/router.svg)](https://pkg.go.dev/jochum.dev/jo-micro/router)

# router

A dynamic router (API Gatway) for go-micro.

It looks for services that host "proto/routerclientpb/routerclientpb.RouterClientService" and ask's them for routes/endpoints, then it registers that endpoints via a proxy method within gin.

## Caveats

- gin doesn't allow to delete routes, so if you want to delete a route you have to restart go-micro/router.

## Usage

docker-compose:

```yaml
services:
  router:
    restart: unless-stopped
    image: docker.io/jomicro/router:0.2.3
    environment:
      - MICRO_TRANSPORT=grpc
      - MICRO_REGISTRY=nats
      - MICRO_REGISTRY_ADDRESS=nats:4222
      - MICRO_BROKER=nats
      - MICRO_BROKER_ADDRESS=nats:4222
      - SERVER_ADDRESS=:8080
      - LOG_LEVEL=info
    ports:
      - 8080:8080
    depends_on:
      - nats
```

See [cmd/microrouterd/plugins.go](cmd/microrouterd/plugins.go) for a list of availabel transports, registries and brokers.

## Todo

- Add (more) examples.
- Add support for Streams / WebSockets.
- Add support for [debug](https://github.com/asim/go-micro/tree/master/debug).
- Maybe add optional support for [auth](https://github.com/asim/go-micro/blob/master/auth/auth.go).

## Integration examples

Have a look at [internalService](https://jochum.dev/jo-micro/router/blob/master/cmd/microrouterd/main.go#L35) or the author's FOSS project [microlobby](https://github.com/pcdummy/microlobby).

Here's some code from the microlobby project

```go
import (
    "jochum.dev/jo-micro/router"
    "github.com/urfave/cli/v2"
    "go-micro.dev/v4"
    "wz2100.net/microlobby/shared/proto/authservicepb/v1"
)

func main() {
    service := micro.NewService()

    service.Init(
        micro.Action(func(c *cli.Context) error {
            s := service.Server()
            r := router.NewHandler(
                config.RouterURI,
                router.NewRoute(
                    router.Method(router.MethodGet),
                    router.Path("/"),
                    router.Endpoint(authservicepb.AuthV1Service.UserList),
                    router.Params("limit", "offset"),
                    router.AuthRequired(),
                ),
                router.NewRoute(
                    router.Method(router.MethodPost),
                    router.Path("/login"),
                    router.Endpoint(authservicepb.AuthV1Service.Login),
                ),
                router.NewRoute(
                    router.Method(router.MethodPost),
                    router.Path("/register"),
                    router.Endpoint(authservicepb.AuthV1Service.Register),
                ),
                router.NewRoute(
                    router.Method(router.MethodPost),
                    router.Path("/refresh"),
                    router.Endpoint(authservicepb.AuthV1Service.Refresh),
                ),
                router.NewRoute(
                    router.Method(router.MethodDelete),
                    router.Path("/:userId"),
                    router.Endpoint(authservicepb.AuthV1Service.UserDelete),
                    router.Params("userId"),
                    router.AuthRequired(),
                ),
                router.NewRoute(
                    router.Method(router.MethodGet),
                    router.Path("/:userId"),
                    router.Endpoint(authservicepb.AuthV1Service.UserDetail),
                    router.Params("userId"),
                    router.AuthRequired(),
                ),
                router.NewRoute(
                    router.Method(router.MethodPut),
                    router.Path("/:userId/roles"),
                    router.Endpoint(authservicepb.AuthV1Service.UserUpdateRoles),
                    router.Params("userId"),
                    router.AuthRequired(),
                ),
            )
            r.RegisterWithServer(s)
        }
    )
}
```

## Developers corner

### Build podman/docker image

#### Prerequesits

- podman
- [Task](https://taskfile.dev/#/installation)

#### Build

```bash
task
```

#### Remove everything

```bash
task rm
```

## Authors

- René Jochum - rene@jochum.dev

## License

Its dual licensed:

- Apache-2.0
- GPL-2.0-or-later
