package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	_ "time/tzdata"

	"go.uber.org/zap"

	"mb2-go/internal/appcfg"
	"mb2-go/internal/cache"
	"mb2-go/internal/config"
	"mb2-go/internal/db"
	"mb2-go/internal/logger"
	"mb2-go/internal/router"
	"mb2-go/internal/web"
)

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load timezone %s: %v\n", cfg.Timezone, err)
		os.Exit(1)
	}
	time.Local = loc

	debug := cfg.Server.Mode != "release"
	logs, err := logger.New(cfg.Log, debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logs.App.Sync()
	defer logs.Access.Sync()

	conn, err := db.Open(cfg.MySQL, logs.App)
	if err != nil {
		logs.App.Fatal("mysql", zap.Error(err))
	}

	fc, err := cache.New(cfg.Cache.Dir)
	if err != nil {
		logs.App.Fatal("cache", zap.Error(err))
	}

	app := &web.App{
		Cfg:    cfg,
		DB:     conn,
		Cache:  fc,
		AppCfg: appcfg.New(conn, logs.App),
		Log:    logs,
		Loc:    loc,
	}

	engine := router.New(app)
	logs.App.Info("server starting", zap.String("addr", cfg.Server.Addr))
	if err := engine.Run(cfg.Server.Addr); err != nil {
		logs.App.Fatal("server", zap.Error(err))
	}
}
