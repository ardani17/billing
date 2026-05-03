module github.com/ispboss/ispboss/services/network-service

go 1.25.0

require (
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/go-playground/validator/v10 v10.30.2
	github.com/go-routeros/routeros/v3 v3.0.1
	github.com/gofiber/fiber/v2 v2.52.6
	github.com/google/uuid v1.6.0
	github.com/gosnmp/gosnmp v1.43.2
	github.com/hibiken/asynq v0.25.1
	github.com/ispboss/ispboss/pkg/auth v0.0.0
	github.com/ispboss/ispboss/pkg/database v0.0.0
	github.com/ispboss/ispboss/pkg/logger v0.0.0
	github.com/ispboss/ispboss/pkg/queue v0.0.0
	github.com/ispboss/ispboss/pkg/tenant v0.0.0
	github.com/jackc/pgx/v5 v5.7.5
	github.com/redis/go-redis/v9 v9.7.3
	github.com/rs/zerolog v1.34.0
	github.com/spf13/viper v1.20.1
	golang.org/x/crypto v0.49.0
	golang.org/x/time v0.15.0
	pgregory.net/rapid v1.2.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/protobuf v1.36.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/ispboss/ispboss/pkg/auth => ../../pkg/auth
	github.com/ispboss/ispboss/pkg/database => ../../pkg/database
	github.com/ispboss/ispboss/pkg/logger => ../../pkg/logger
	github.com/ispboss/ispboss/pkg/queue => ../../pkg/queue
	github.com/ispboss/ispboss/pkg/tenant => ../../pkg/tenant
)
