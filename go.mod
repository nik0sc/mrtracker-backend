module go.lepak.sg/mrtracker-backend

// +heroku goVersion go1.17
// +heroku install ./cmd/server/...
go 1.17

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/prometheus/client_golang v1.11.0
)
