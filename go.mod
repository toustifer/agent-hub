module github.com/stifer/agent-hub

go 1.26.1

require (
	entgo.io/ent v0.14.5
	github.com/Wei-Shaw/sub2api v0.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.6.0
	github.com/redis/go-redis/v9 v9.6.1
	github.com/spf13/viper v1.19.0
	go.uber.org/zap v1.27.0
)

// 通过 replace 复用 sub2api v0.1.104 的包
// 部署时需先把 sub2api 源码放到 /opt/sub2api-src/backend
// 本地开发时改成本地路径
replace github.com/Wei-Shaw/sub2api => /opt/sub2api-src/backend
