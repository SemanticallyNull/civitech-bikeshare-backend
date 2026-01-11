package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kong"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/semanticallynull/bookingengine-backend/api"
	"github.com/semanticallynull/bookingengine-backend/bike"
	"github.com/semanticallynull/bookingengine-backend/internal/o11y"
	"github.com/semanticallynull/bookingengine-backend/station"
)

var cli = struct {
	DatabaseURL string `name:"database-url" env:"DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"` //nolint:lll
	Port        int    `name:"port" env:"PORT" default:"8080"`

	Auth0Domain string `name:"auth0-domain" env:"AUTH0_DOMAIN"`
	Audience    string `name:"audience" env:"AUDIENCE"`

	MetricsUsername string `name:"metrics-username" env:"METRICS_USERNAME"`
	MetricsPassword string `name:"metrics-password" env:"METRICS_PASSWORD"`
}{}

func main() {
	if err := run(); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	kong.Parse(&cli)

	db, err := sqlx.ConnectContext(ctx, "pgx",
		cli.DatabaseURL)
	if err != nil {
		return err
	}
	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	br := bike.NewRepository(db)
	sr := station.NewRepository(db)

	obs, cleanup, err := o11y.Setup(ctx)
	defer cleanup()
	if err != nil {
		return err
	}

	a := api.New(br, sr, obs, cli.Auth0Domain, cli.Audience, cli.MetricsUsername, cli.MetricsPassword)

	serv := http.Server{
		Addr:    fmt.Sprintf(":%d", cli.Port),
		Handler: a.Router(),
	}

	go func() {
		if err := serv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = serv.Shutdown(ctx)
	if err != nil {
		return err
	}
	return nil
}
