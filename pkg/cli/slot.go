// Copyright 2014 Wandoujia Inc. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/IceFireDB/kit/pkg/models"
	"github.com/juju/errors"

	log "github.com/IceFireDB/kit/pkg/logger"
	uuid "github.com/nu7hatch/gouuid"
)

var (
	store       *models.Store
	productName string
	slotNum     int
)

func NewSlotCmd() *cli.Command {
	c := &cli.Command{
		Name: "slot",
		Subcommands: []*cli.Command{
			{
				Name:        "init",
				Description: "init slot",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "force set slot to initial state regardless of existence",
						Value:   false,
					},
				},
				Action: runSlotInit,
			},
			{
				Name:        "info",
				Description: "show slot info",
				Action:      runSlotInfo,
			},
			{
				Name:        "set",
				Description: "set <slot_id> <group_id> <status>",
				Action:      runSlotSet,
			},
			{
				Name:        "range-set",
				Description: "range-set <slot_from> <slot_to> <group_id> <status>",
				Action:      runSlotRangeSet,
			},
			{
				Name:        "migrate",
				Description: "migrate <slot_from> <slot_to> <group_id>",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "delay",
						Usage: "delay time in ms",
					},
				},
				Action: runSlotMigrate,
			},
		},
		Before: func(c *cli.Context) error {
			store = c.Context.Value("store").(*models.Store)
			productName = c.Context.Value("product").(string)
			slotNum = c.Context.Value("slotNum").(int)
			return nil
		},
	}
	return c
}

func runSlotInit(context *cli.Context) error {
	isForce := context.Bool("force")
	if !isForce {
		s, err := store.GetSlot(0, true)
		if err != nil {
			return errors.Trace(err)
		}
		if s != nil {
			return errors.New("slots already exists. use -f flag to force init")
		}
	}
	err := store.InitSlotSet(productName, slotNum)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func runSlotInfo(context *cli.Context) error {
	slotId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return err
	}
	s, err := store.GetSlot(slotId, true)
	if err != nil {
		return errors.Trace(err)
	}
	b, _ := json.MarshalIndent(s, " ", "  ")
	fmt.Println(string(b))
	return nil
}

func runSlotRangeSet(context *cli.Context) error {
	fromSlotId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return fmt.Errorf("parse fromSlotId err %w", err)
	}
	toSlotId, err := strconv.Atoi(context.Args().Get(1))
	if err != nil {
		return fmt.Errorf("parse toSlotId err %w", err)
	}
	groupId, err := strconv.Atoi(context.Args().Get(2))
	if err != nil {
		return fmt.Errorf("parse groupId err %w", err)
	}
	status := context.Args().Get(3)
	err = store.SetSlotRange(productName, fromSlotId, toSlotId, groupId, models.SlotStatus(status))
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func runSlotSet(context *cli.Context) error {
	slotId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return fmt.Errorf("parse slotId err %w", err)
	}
	groupId, err := strconv.Atoi(context.Args().Get(1))
	if err != nil {
		return fmt.Errorf("parse groupId err %w", err)
	}
	status := context.Args().Get(2)
	err = store.SetSlotRange(productName, slotId, slotId, groupId, models.SlotStatus(status))
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func runSlotMigrate(context *cli.Context) error {
	fromSlotId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return fmt.Errorf("parse fromSlotId err %w", err)
	}
	toSlotId, err := strconv.Atoi(context.Args().Get(1))
	if err != nil {
		return fmt.Errorf("parse toSlotId err %w", err)
	}
	newGroupId, err := strconv.Atoi(context.Args().Get(2))
	if err != nil {
		return fmt.Errorf("parse groupId err %w", err)
	}
	delay := context.Int("delay")
	t := &MigrateTask{}
	t.Delay = delay
	t.FromSlot = fromSlotId
	t.ToSlot = toSlotId
	t.NewGroupId = newGroupId
	t.Status = "migrating"
	t.CreateAt = strconv.FormatInt(time.Now().Unix(), 10)
	u, err := uuid.NewV4()
	if err != nil {
		log.Warn(err)
		return errors.Trace(err)
	}
	t.Id = u.String()
	t.stopChan = make(chan struct{})

	// run migrate
	if ok, err := preMigrateCheck(t); ok {
		err = RunMigrateTask(t)
		if err != nil {
			log.Warn(err)
			return errors.Trace(err)
		}
	} else {
		log.Warn(err)
		return errors.Trace(err)
	}
	return nil
}
