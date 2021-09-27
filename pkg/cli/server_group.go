// Copyright 2014 Wandoujia Inc. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/juju/errors"

	"github.com/urfave/cli/v2"

	"github.com/IceFireDB/kit/pkg/models"

	log "github.com/IceFireDB/kit/pkg/logger"
)

// codis redis instance manage tool

func NewGroupCmd() *cli.Command {
	c := &cli.Command{
		Name: "server",
		Subcommands: []*cli.Command{
			{
				Name:        "list",
				Description: "list server groups",
				Action:      runListServerGroup,
			},
			{
				Name:        "add",
				Description: "add <group_id> <redis_addr>",
				Action:      runAddServerToGroup,
			},
			{
				Name:        "remove",
				Description: "remove <group_id> <redis_addr>",
				Action:      runRemoveServerFromGroup,
			},
			{
				Name:        "remove-group",
				Description: "remove-group <group_id>",
				Action:      runRemoveServerGroup,
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

func runAddServerToGroup(context *cli.Context) error {
	groupId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return err
	}
	addr := context.Args().Get(1)
	serverGroup, err := store.LoadGroup(groupId, false)
	if err != nil {
		return err
	}
	if serverGroup == nil || serverGroup.Id == 0 {
		serverGroup = models.NewServerGroup(productName, groupId)
	}
	if len(addr) == 0 {
		return errors.New("data node addr is required")
	}
	exists, err := serverGroup.ServerExists(addr)
	if err != nil {
		return err
	}
	// if server exist in group ignore
	if exists {
		return nil
	}

	server, err := store.GetServer(addr, true)
	if err != nil {
		return err
	}
	serverGroup.Servers = append(serverGroup.Servers, *server)
	err = store.UpdateGroup(serverGroup)
	if err != nil {
		return err
	}
	return nil
}

func runListServerGroup(context *cli.Context) error {
	groups, err := store.ListGroup()
	if err != nil {
		log.Warn(err)
		return err
	}
	b, _ := json.MarshalIndent(groups, " ", "  ")
	fmt.Println(string(b))
	return nil
}

func runRemoveServerGroup(context *cli.Context) error {
	groupId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return err
	}
	sg, err := store.LoadGroup(groupId, true)
	if err != nil {
		return err
	}

	if len(sg.Servers) != 0 {
		for _, server := range sg.Servers {
			err := store.DeleteServer(server.Addr)
			if err != nil {
				// todo check if not exit if zookeeper will return err
				log.Error(err)
				return err
			}
		}
	}
	err = store.DeleteGroup(groupId)
	if err != nil {
		return err
	}

	return nil
}

func runRemoveServerFromGroup(context *cli.Context) error {
	groupId, err := strconv.Atoi(context.Args().Get(0))
	if err != nil {
		return err
	}
	addr := context.Args().Get(1)
	serverGroup, err := store.LoadGroup(groupId, true)
	if err != nil {
		log.Warn(err)
		return err
	}
	if len(serverGroup.Servers) == 0 {
		return errors.New("group has no server")
	}
	servers := make([]models.Server, 0, len(serverGroup.Servers)-1)
	for _, s := range serverGroup.Servers {
		if s.Addr == addr {
			continue
		}
		servers = append(servers, s)
	}
	serverGroup.Servers = servers
	if err := store.UpdateGroup(serverGroup); err != nil {
		return err
	}
	err = store.DeleteServer(addr)
	if err != nil {
		return err
	}
	return nil
}
