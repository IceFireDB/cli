// Copyright 2014 Wandoujia Inc. All Rights Reserved.
// Licensed under the MIT (MIT-LICENSE.txt) license.

package cli

import (
	"container/list"
	"sync"

	"github.com/IceFireDB/kit/pkg/models"
	"github.com/juju/errors"

	log "github.com/IceFireDB/kit/pkg/logger"
)

var (
	pendingMigrateTask = list.New()
	curMigrateTask     *MigrateTask
	lck                = sync.RWMutex{}
)

const (
	MIGRATE_TASK_PENDING   string = "pending"
	MIGRATE_TASK_MIGRATING string = "migrating"
	MIGRATE_TASK_FINISHED  string = "finished"
	MIGRATE_TASK_ERR       string = "error"
)

type MigrateTaskForm struct {
	FromSlot   int    `json:"from"`
	ToSlot     int    `json:"to"`
	NewGroupId int    `json:"new_group"`
	Delay      int    `json:"delay"`
	CreateAt   string `json:"create_at"`
	Percent    int    `json:"percent"`
	Status     string `json:"status"`
	Id         string `json:"id"`
}

type MigrateTask struct {
	MigrateTaskForm

	stopChan chan struct{}
}

func findPendingMigrateTask(id string) *MigrateTask {
	for e := pendingMigrateTask.Front(); e != nil; e = e.Next() {
		t := e.Value.(*MigrateTask)
		if t.Id == id {
			return t
		}
	}
	return nil
}

func removePendingMigrateTask(id string) bool {
	for e := pendingMigrateTask.Front(); e != nil; e = e.Next() {
		t := e.Value.(*MigrateTask)
		if t.Id == id && t.Status == "pending" {
			pendingMigrateTask.Remove(e)
			return true
		}
	}
	return false
}

// migrate multi slots
func RunMigrateTask(task *MigrateTask) error {
	err := store.Lock()
	if err != nil {
		return err
	}
	defer func() {
		_ = store.UnLock()
	}()

	to := task.NewGroupId
	task.Status = MIGRATE_TASK_MIGRATING
	for slotId := task.FromSlot; slotId <= task.ToSlot; slotId++ {
		err := func() error {
			log.Info("start migrate slot:", slotId)

			// todo lock for migrate single slot
			// set slot status
			s, err := store.GetSlot(slotId, true)
			if err != nil {
				log.Error(err)
				return err
			}
			if s.State.Status != models.SLOT_STATUS_ONLINE && s.State.Status != models.SLOT_STATUS_MIGRATE {
				log.Warn("status is not online && migrate", s)
				return nil
			}

			from := s.GroupId
			if s.State.Status == models.SLOT_STATUS_MIGRATE {
				from = s.State.MigrateStatus.From
			}

			// make sure from group & target group exists
			exists, err := store.GroupExists(from)
			if err != nil {
				return errors.Trace(err)
			}
			if !exists {
				log.Errorf("src group %d not exist when migrate from %d to %d", from, from, to)
				return errors.NotFoundf("group %d", from)
			}
			exists, err = store.GroupExists(to)
			if err != nil {
				return errors.Trace(err)
			}
			if !exists {
				return errors.NotFoundf("group %d", to)
			}

			// cannot migrate to itself
			if from == to {
				log.Warn("from == to, ignore", s)
				return nil
			}

			// modify slot status
			if err := store.SetMigrateStatus(s, from, to); err != nil {
				log.Error(err)
				return err
			}

			// do real migrate
			err = MigrateSingleSlot(slotId, from, to, task.Delay, task.stopChan)
			if err != nil {
				log.Error(err)
				return err
			}

			// migrate done, change slot status back
			s.State.Status = models.SLOT_STATUS_ONLINE
			s.State.MigrateStatus.From = models.INVALID_ID
			s.State.MigrateStatus.To = models.INVALID_ID
			if err := store.UpdateSlot(s); err != nil {
				log.Error(err)
				return err
			}
			return nil
		}()
		if err == ErrStopMigrateByUser {
			log.Info("stop migration job by user")
			break
		} else if err != nil {
			log.Error(err)
			task.Status = MIGRATE_TASK_ERR
			return err
		}
		task.Percent = (slotId - task.FromSlot + 1) * 100 / (task.ToSlot - task.FromSlot + 1)
		log.Info("total percent:", task.Percent)
	}
	task.Status = MIGRATE_TASK_FINISHED
	log.Info("migration finished")
	return nil
}

func preMigrateCheck(t *MigrateTask) (bool, error) {
	slots, err := store.GetMigratingSlots()
	if err != nil {
		return false, err
	}
	// check if there is migrating slot
	if len(slots) == 0 {
		return true, nil
	} else if len(slots) > 1 {
		return false, errors.New("more than one slots are migrating, unknown error")
	} else if len(slots) == 1 {
		slot := slots[0]
		if t.NewGroupId != slot.State.MigrateStatus.To || t.FromSlot != slot.Id || t.ToSlot != slot.Id {
			return false, errors.Errorf("there is a migrating slot %+v, finish it first", slot)
		}
	}
	return true, nil
}

/*func migrateTaskWorker() {
	for {
		select {
		case <-time.After(1 * time.Second):
			{
				// check if there is new task
				lck.RLock()
				cnt := pendingMigrateTask.Len()
				lck.RUnlock()
				if cnt > 0 {
					lck.RLock()
					t := pendingMigrateTask.Front()
					lck.RUnlock()

					log.Info("new migrate task arrive")
					if t != nil {
						lck.Lock()
						curMigrateTask = t.Value.(*MigrateTask)
						lck.Unlock()

						if ok, err := preMigrateCheck(curMigrateTask); ok {
							RunMigrateTask(curMigrateTask)
						} else {
							log.Warn(err)
						}

						lck.Lock()
						curMigrateTask = nil
						lck.Unlock()
					}
					log.Info("migrate task", t, "done")

					lck.Lock()
					if t != nil {
						pendingMigrateTask.Remove(t)
					}
					lck.Unlock()
				}
			}
		}
	}
}*/
