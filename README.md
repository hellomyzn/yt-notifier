# docker-go

Docker で Go アプリケーションを開発・実行するためのスタータープロジェクトです。

- [setup](#setup)
- [docker commands](#docker-commands)
- [start](#start)
- [containers](#containers)
- [project-structure](#project-structure)

### setup
```bash
cp .env.template .env
```
必要に応じてポート番号などを調整してください。

### docker commands
```bash
# build & up container
docker compose up -d --build
$ make up
# destroy
docker compose down --rmi all --volumes --remove-orphans
$ make destroy
# attach to go container (docker compose exec go bash)
$ make go
# down container
docker compose down
$ make down
```

### start
1. run container
```bash
$ make up
```
2. check http server (default port: 8080)
```bash
$ curl http://localhost:8080
$ curl http://localhost:8080/healthz
```

### containers
- go
  - Go アプリケーション実行用のコンテナ

### project-structure
```
/project_root
├── infra/docker/go/      # Go 用 Dockerfile
├── src/                  # Go アプリケーションのソースコード
│   ├── cmd/server/main.go
│   └── go.mod
├── docker-compose.yml
├── Makefile
└── README.md (this file)
```
