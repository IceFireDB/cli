// Copyright 2014 Wandoujia Inc. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkgcli "github.com/IceFireDB/cli/pkg/cli"
	"github.com/IceFireDB/kit/pkg/models"
	"golang.org/x/net/context"

	"github.com/urfave/cli/v2"

	"github.com/ledisdb/xcodis/utils"
	_ "net/http/pprof"

	"github.com/c4pt0r/cfg"
	"github.com/juju/errors"
	log "github.com/ngaut/logging"
)

// build info
var (
	BuildVersion = "unknown"
	BuildDate    = "unknown"
)

// global objects
var (
	productName string
	config      *cfg.Cfg
	livingNode  string
	broker      = "ledisdb"
	slotNum     = 128
	store       *models.Store
)

type Command struct {
	Run   func(cmd *Command, args []string)
	Usage string
	Short string
	Long  string
	Flag  flag.FlagSet
	Ctx   interface{}
}

func registerConfigNode() error {
	lock := models.NewLock()

	err := store.RegisterActiveCli(lock)
	if err != nil {
		return errors.Trace(err)
	}

	livingNode = lock.Name()

	return nil
}

func unRegisterConfigNode() {
	log.Debugf("unRegisterConfigNode %s", livingNode)
	if len(livingNode) > 0 {
		_ = store.UnregisterActiveCli(livingNode)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "pd"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    "init config file",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "log-file",
			Aliases:  []string{"L"},
			Usage:    "log file",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "log-level",
			Aliases:  []string{"l"},
			Usage:    "log level",
			Required: false,
		},
		&cli.StringFlag{
			Name: "broker",
		},
		&cli.StringFlag{
			Name: "slot-num",
		},
	}
	app.Commands = []*cli.Command{pkgcli.NewSlotCmd(), pkgcli.NewGroupCmd()}
	app.Before = func(ctx *cli.Context) (err error) {
		log.SetLevelByString("info")

		configFile := ctx.String("config")
		config, err = utils.InitConfigFromFile(configFile)
		if err != nil {
			panic(err)
		}
		if logfile := ctx.String("log-file"); logfile != "" {
			log.SetOutputByName(ctx.String("log-file"))
		}
		if logLevel := ctx.String("log-file"); logLevel != "" {
			log.SetLevelByString(logLevel)
		}

		coordinatorType, _ := config.ReadString("coordinator_type", "etcd")
		coordinatorAddr, _ := config.ReadString("coordinator_addr", "localhost:2379")
		productName, _ = config.ReadString("product", "test")
		ctx.Context = context.WithValue(ctx.Context, "product", productName)
		client, err := models.NewClient(coordinatorType, coordinatorAddr, "", time.Second*5)
		if err != nil {
			panic(err)
		}
		store = models.NewStore(client, productName)
		ctx.Context = context.WithValue(ctx.Context, "store", store)
		broker, _ = config.ReadString("broker", "redis")
		slotNum, _ = config.ReadInt("slot_num", 128)
		ctx.Context = context.WithValue(ctx.Context, "slotNum", slotNum)

		log.Debugf("product: %s", productName)
		log.Debugf("broker: %s", broker)

		if err := registerConfigNode(); err != nil {
			log.Fatal(errors.ErrorStack(err))
		}

		//if err := removeOrphanLocks(); err != nil {
		//	log.Fatal(errors.ErrorStack(err))
		//}

		return nil
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(0)
	}()

	defer func() {
		unRegisterConfigNode()
	}()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
