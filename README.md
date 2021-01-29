# tipstocks
Simple CR(U)D App for managing your favorite links!

# Build & Run
App will be built & run as docker-compose

## \[Way 1\] simple setup

```bash
# as foreground
$ ./run.sh --build

# as background
# $ ./run.sh --build -d
```

## \[Way 2\] run without building container image

```bash
# as foreground
$ ./run.sh

# as background
# $ ./run.sh -d
```

# Testing
testing functions with running the gRPC server of app

### 1. setup mongoDB

```bash
[terminal1](tipstocks)$ brew tap mongodb/brew
[terminal1](tipstocks)$ brew install mongodb-community@4.4
[terminal1](tipstocks)$ brew services start mongodb-community@4.4
```

### 2. run gRPC server

```bash
[terminal1](tipstocks)$ cd app/server
[terminal1](tipstocks/app/server)$ go run server.go
```

### 3. test (using another tab or window as "terminal2")

```bash
[terminal2](tipstocks)$ go test -v ./...
```

### 4. terminate server (on terminal1)

```bash
[terminal1] server running ...
# press [Control + C]
```

### 5. terminate DB (on terminal2)
```bash
[terminal2](tipstocks)$ brew services stop mongodb-community@4.4
```
