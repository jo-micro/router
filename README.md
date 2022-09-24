[![Build Status](https://drone.fk.jochum.dev/api/badges/jo-micro/router/status.svg)](https://drone.fk.jochum.dev/jo-micro/router) [![Go Reference](https://pkg.go.dev/badge/jochum.dev/jo-micro/router.svg)](https://pkg.go.dev/jochum.dev/jo-micro/router)

# router

A dynamic router (API Gatway) for go-micro.

It looks for services that host "internal/proto/routerclientpb/routerclientpb.RouterClientService" and ask's them for routes/endpoints, then it registers that endpoints via a proxy method within gin.

## Caveats

- gin doesn't allow to delete routes, so if you want to delete a route you have to restart go-micro/router.

## Usage

### docker-compose without auth

docker-compose:

```yaml
services:
  router:
    restart: unless-stopped
    image: docker.io/jomicro/router:0.3.8
    environment:
      - MICRO_TRANSPORT=grpc
      - MICRO_REGISTRY=nats
      - MICRO_REGISTRY_ADDRESS=nats:4222
      - MICRO_BROKER=nats
      - MICRO_BROKER_ADDRESS=nats:4222
      - MICRO_ROUTER_LISTEN=:8080
      - MICRO_ROUTER_LOG_LEVEL=info

    ports:
      - 8080:8080
    depends_on:
      - nats
```

### docker-compose with auth

```yaml
services:
  router:
    restart: unless-stopped
    image: docker.io/jomicro/router:0.3.8
    environment:
      - MICRO_AUTH2_CLIENT=jwt
      - MICRO_AUTH2_ROUTER=jwt
      - MICRO_AUTH2_JWT_AUDIENCES="key from task keys inside the auth project"
      - MICRO_AUTH2_JWT_PRIV_KEY="key from task keys inside the auth project"
      - MICRO_AUTH2_JWT_PUB_KEY="key from task keys inside the auth project"
      - MICRO_TRANSPORT=grpc
      - MICRO_REGISTRY=nats
      - MICRO_REGISTRY_ADDRESS=nats:4222
      - MICRO_BROKER=nats
      - MICRO_BROKER_ADDRESS=nats:4222
      - MICRO_ROUTER_LISTEN=:8080
      - MICRO_ROUTER_LOG_LEVEL=info

    ports:
      - 8080:8080
    depends_on:
      - nats
```

See [cmd/microrouterd/plugins.go](cmd/microrouterd/plugins.go) for a list of availabel transports, registries and brokers.

## Todo

- Add support for Streams / WebSockets.
- Add support for [debug](https://github.com/asim/go-micro/tree/master/debug)?

## Service integration examples

Have a look at [internalService](cmd/microrouterd/main.go#L51 ), [jo-micro/auth2](https://git.jochum.dev/jo-micro/auth2/src/branch/main/cmd/microauth2sqld/main.go#L319) or the author's FOSS project [microlobby](https://github.com/pcdummy/microlobby).

Here's some code from the jo-micro/auth2 project

```go
import (
    "jochum.dev/jo-micro/router"
    "github.com/urfave/cli/v2"
    "go-micro.dev/v4"
    "jochum.dev/jo-micro/auth2/internal/proto/authpb"
)

func main() {
    service := micro.NewService()

    service.Init(
        micro.Action(func(c *cli.Context) error {
            s := service.Server()
            r := router.NewHandler(
              "api/auth/v1",
              router.NewRoute(
                router.Method(router.MethodGet),
                router.Path("/"),
                router.Endpoint(authpb.AuthService.List),
                router.Params("limit", "offset"),
                router.AuthRequired(),
                router.RatelimitUser("1-S", "10-M"),
              ),
              router.NewRoute(
                router.Method(router.MethodPost),
                router.Path("/login"),
                router.Endpoint(authpb.AuthService.Login),
                router.RatelimitClientIP("1-S", "10-M", "30-H", "100-D"),
              ),
              router.NewRoute(
                router.Method(router.MethodPost),
                router.Path("/register"),
                router.Endpoint(authpb.AuthService.Register),
                router.RatelimitClientIP("1-M", "10-H", "50-D"),
              ),
              router.NewRoute(
                router.Method(router.MethodPost),
                router.Path("/refresh"),
                router.Endpoint(authpb.AuthService.Refresh),
                router.RatelimitClientIP("1-M", "10-H", "50-D"),
              ),
              router.NewRoute(
                router.Method(router.MethodDelete),
                router.Path("/:userId"),
                router.Endpoint(authpb.AuthService.Delete),
                router.Params("userId"),
                router.AuthRequired(),
                router.RatelimitUser("1-S", "10-M"),
              ),
              router.NewRoute(
                router.Method(router.MethodGet),
                router.Path("/:userId"),
                router.Endpoint(authpb.AuthService.Detail),
                router.Params("userId"),
                router.AuthRequired(),
                router.RatelimitUser("100-M"),
              ),
              router.NewRoute(
                router.Method(router.MethodPut),
                router.Path("/:userId/roles"),
                router.Endpoint(authpb.AuthService.UpdateRoles),
                router.Params("userId"),
                router.AuthRequired(),
                router.RatelimitUser("1-M"),
              ),
            )
            r.RegisterWithServer(s)
        })
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
cp .env.sample .env
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
