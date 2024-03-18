package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/sjmudd/stopwatch"
)

const (
	DBUS_OBJ_PATH      = "/org/freedesktop/systemd1"
	DBUS_MANAGER_IFACE = "org.freedesktop.systemd1.Manager"
	DBUS_JOB_IFACE     = "org.freedesktop.systemd1.Job"
	DBUS_GET_JOBS      = "org.freedesktop.systemd1.Manager.ListJobs"
	DBUS_NEW_JOB       = "org.freedesktop.systemd1.Manager.JobNew"
	DBUS_DEL_JOB       = "org.freedesktop.systemd1.Manager.JobRemoved"
)

var job_map map[uint32]Job = make(map[uint32]Job)
var job_map_mutex sync.RWMutex = sync.RWMutex{}

var newJobSignal chan *dbus.Signal = make(chan *dbus.Signal, 10)

type Job struct {
	watch *stopwatch.Stopwatch
	jID   uint32
	jPath dbus.ObjectPath
	jType string

	jBefore uint32
	jAfter  uint32

	sUnit string
	sPath dbus.ObjectPath
}

type DBUS_SYSTEMD_JOB struct {
	id     uint32
	path   dbus.ObjectPath
	unit   string
	result string
}

func initSystemdJobStuff(conn *dbus.Conn) error {
	job_map[0] = Job{}
	fmt.Println("initSystemdJobStuff()")
	fmt.Println(job_map)

	err := getAllJobs(conn)
	if err != nil {
		return err
	}

	err = conn.AddMatchSignal(
		dbus.WithMatchInterface(DBUS_MANAGER_IFACE),
		dbus.WithMatchObjectPath(DBUS_OBJ_PATH),
	)
	if err != nil {
		return err
	}

	conn.Signal(newJobSignal)
	go processDbusSignals(conn)

	return nil
}

func getAllJobs(conn *dbus.Conn) error {
	var arr []interface{}
	conn.BusObject()
	obj := conn.Object("org.freedesktop.systemd1", DBUS_OBJ_PATH)
	feat, err1 := obj.GetProperty(DBUS_MANAGER_IFACE + ".Features")

	if err1 != nil {
		fmt.Println(err1)
		return err1
	}
	fmt.Printf("Systemd Features: %v\n", feat)
	err := obj.Call(DBUS_GET_JOBS, 0).Store(&arr)
	fmt.Printf("ListJobs(): %+v\n", arr)

	job_map_mutex.Lock()
	defer job_map_mutex.Unlock()

	for _, iface := range arr {
		var i = iface.([]interface{})
		jID := i[0].(uint32)
		job_map[jID] = Job{
			watch:   stopwatch.Start(stopwatch.DefaultFormat),
			jID:     jID,
			jPath:   i[4].(dbus.ObjectPath),
			jType:   i[2].(string),
			jBefore: 0,
			jAfter:  0,
			sUnit:   i[1].(string),
			sPath:   i[5].(dbus.ObjectPath),
		}
		fmt.Println(jID)
		fmt.Printf("map: %+v\n", job_map)
	}
	return err
}

func processDbusSignals(conn *dbus.Conn) {
	for {
		fmt.Println("Wait for dbus signal...")
		sig, ok := <-newJobSignal
		if !ok {
			return
		}
		job_map_mutex.Lock()
		if sig.Name == DBUS_NEW_JOB {
			job := Job{
				watch:   stopwatch.Start(stopwatch.DefaultFormat),
				jID:     sig.Body[0].(uint32),
				jPath:   sig.Body[1].(dbus.ObjectPath),
				jType:   "",
				jBefore: 0,
				jAfter:  0,
				sUnit:   sig.Body[2].(string),
				sPath:   DBUS_OBJ_PATH,
			}

			obj := conn.Object("org.freedesktop.systemd1", job.jPath)
			feat, err := obj.GetProperty(DBUS_JOB_IFACE + ".Unit")
			if err == nil {
				job.sPath = feat.Value().([]interface{})[1].(dbus.ObjectPath)
			}
			job_map[sig.Body[0].(uint32)] = job

			fmt.Printf("NewJob: %+v\n", job)
		}
		if sig.Name == DBUS_DEL_JOB {
			fmt.Println(sig.Name)
			fmt.Println(sig.Sender)
			body := DBUS_SYSTEMD_JOB{
				id:     sig.Body[0].(uint32),
				path:   sig.Body[1].(dbus.ObjectPath),
				unit:   sig.Body[2].(string),
				result: sig.Body[3].(string),
			}
			fmt.Println(body)
			delete(job_map, body.id)
		}
		job_map_mutex.Unlock()
	}
}

func getJobs(conn *dbus.Conn) map[uint32]Job {
	fmt.Println("getJobs()")
	job_map_mutex.Lock()
	defer job_map_mutex.Unlock()

	job_map = make(map[uint32]Job)
	getAllJobs(conn)

	return job_map
}

func getOldestJob() (Job, error) {
	var longest_id uint32 = 0
	var longest_elapsed time.Duration = 0

	job_map_mutex.Lock()
	defer job_map_mutex.Unlock()

	if len(job_map) < 1 {
		return Job{}, errors.New("currently no jobs running")
	}

	for job_id, job := range job_map {
		if job_id > 0 && job.watch.Elapsed() > longest_elapsed {
			longest_id = job_id
		}
	}
	return job_map[longest_id], nil
}
