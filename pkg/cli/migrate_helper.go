// Copyright 2014 Wandoujia Inc. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IceFireDB/kit/pkg/models"
	"golang.org/x/net/context"

	log "github.com/IceFireDB/kit/pkg/logger"

	"github.com/garyburd/redigo/redis"
	_ "github.com/juju/errors"
)

const (
	MIGRATE_TIMEOUT = 30000
)

// ErrGroupMasterNotFound = errors.New("group master not found")
var ErrInvalidAddr = errors.New("invalid addr")

type migrater struct {
	group string
}

func (m *migrater) nextGroup() {
	switch m.group {
	case "KV":
		m.group = "HASH"
	case "HASH":
		m.group = "LIST"
	case "LIST":
		m.group = "SET"
	case "SET":
		m.group = "ZSET"
	case "ZSET":
		m.group = ""
	}
}

// return: success_count, remain_count, error
// slotsmgrt host port timeout slotnum count
func (m *migrater) sendRedisMigrateCmd(c redis.Conn, slotId int, toAddr string) (bool, error) {
	addrParts := strings.Split(toAddr, ":")
	if len(addrParts) != 2 {
		return false, ErrInvalidAddr
	}

	// use scan and migrate
	reply, err := redis.MultiBulk(c.Do("scan", 0))
	if err != nil {
		return false, err
	}

	var next string
	var keys []interface{}

	if _, err := redis.Scan(reply, &next, &keys); err != nil {
		return false, err
	}

	for _, key := range keys {
		if _, err := c.Do("migrate", addrParts[0], addrParts[1], key, slotId, MIGRATE_TIMEOUT); err != nil {
			// todo, try del if key exists
			return false, err
		}
	}

	return next != "0", nil
}

func (m *migrater) sendLedisMigrateCmd(c redis.Conn, slotId int, toAddr string) (bool, error) {
	addrParts := strings.Split(toAddr, ":")
	if len(addrParts) != 2 {
		return false, ErrInvalidAddr
	}

	count := 10
	num, err := redis.Int(c.Do("migratedb", addrParts[0], addrParts[1], m.group, count, slotId, MIGRATE_TIMEOUT))
	if err != nil {
		return false, err
	} else if num < count {
		m.nextGroup()
		return m.group != "", nil
	} else {
		return true, nil
	}
}

func (m *migrater) sendMigrateCmd(c redis.Conn, slotId int, toAddr string) (bool, error) {
	//if broker == LedisBroker {
	//	return m.sendLedisMigrateCmd(c, slotId, toAddr)
	//} else {
	//return m.sendRedisMigrateCmd(c, slotId, toAddr)
	//}
	return m.sendLedisMigrateCmd(c, slotId, toAddr)
}

var ErrStopMigrateByUser = errors.New("migration stop by user")

func MigrateSingleSlot(slotId, fromGroup, toGroup int, delay int, stopChan <-chan struct{}) error {
	groupFrom, err := store.LoadGroup(fromGroup, true)
	if err != nil {
		return fmt.Errorf("load from group err %w", err)
	}
	groupTo, err := store.LoadGroup(toGroup, true)
	if err != nil {
		return fmt.Errorf("load to group err %w", err)
	}

	var fromMaster, toMaster *models.Server

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	for fromMaster == nil || toMaster == nil {
		if ctx.Err() != nil {
			break
		}
		if fromMaster == nil {
			fromMaster, err = store.Master(groupFrom)
		}
		if toMaster == nil {
			toMaster, err = store.Master(groupTo)
		}

		//toMaster, err = store.Master(groupTo)
	}
	if err != nil {
		return err
	}

	if fromMaster == nil || toMaster == nil {
		return models.ErrGroupMasterNotFound
	}

	c, err := redis.Dial("tcp", fromMaster.Addr)
	if err != nil {
		return err
	}

	defer c.Close()

	m := new(migrater)
	m.group = "KV"

	remain, err := m.sendMigrateCmd(c, slotId, toMaster.Addr)
	if err != nil {
		return err
	}

	num := 0
	for remain {
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		if stopChan != nil {
			select {
			case <-stopChan:
				return ErrStopMigrateByUser
			default:
			}
		}
		remain, err = m.sendMigrateCmd(c, slotId, toMaster.Addr)
		if num%500 == 0 && remain {
			log.Infof("still migrating")
		}
		num++
		if err != nil {
			return err
		}
	}
	return nil
}
