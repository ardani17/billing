module github.com/ispboss/ispboss/pkg/tenant

go 1.24

require (
	github.com/gofiber/fiber/v2 v2.52.6
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/ispboss/ispboss/pkg/auth v0.0.0
)

replace github.com/ispboss/ispboss/pkg/auth => ../auth
