# router

A dynamic router (API Gatway) for go-micro.

It looks for services that host "proto/routerclientpb/routerclientpb.RouterClient" and ask's them for routes/endpoints,
then it registers that endpoints via a proxy method within gin.

## Caveats

- gin doesn't allow to delete routes, so if you want to delete a route you have to restart go-micro/router.

## Build podman/docker image

### Prerequesits

- podman
- [Task](https://taskfile.dev/#/installation)

### Build

```bash
task
```

### Remove everything except the result created by task

```bash
task rm
```

## Authors

- René Jochum - rene@jochum.dev

## License

Its dual licensed:

- Apache-2.0
- GPL-2.0-or-later
