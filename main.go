package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/hairyhenderson/go-which"
)

func main() {
	fmt.Println("Starting...")

	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		fmt.Println("dbus systembus not available!")
		os.Exit(1)
	}
	defer conn.Close()
	if err = conn.Auth(nil); err != nil {
		panic(err)
	}

	if err = conn.Hello(); err != nil {
		panic(err)
	}

	if err = initSystemdJobStuff(conn); err != nil {
		panic(err)
	}

	go processJobs(conn)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}

func processJobs(conn *dbus.Conn) {
	fmt.Println("Start timeloop...")

	wasDisplaying := false
	var exePath string = ""

	for range time.Tick(time.Second * 1) {
		if exePath == "" {
			exePath = which.Which("plymouth")
			fmt.Printf("ExecPath=%s\n", exePath)
			continue
		}
		current_jobs := getOldestJob(conn)
		if current_jobs.jID > 0 && current_jobs.watch.ElapsedSeconds() > 1.0 {
			fmt.Println(current_jobs)
			txtArg := fmt.Sprintf("--text=\"Job: Unit: %s\" Time:%f ", current_jobs.sUnit, current_jobs.watch.ElapsedSeconds())
			exe := exec.Command(exePath, "display-message", txtArg)
			_, err := exe.Output()
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					exePath = ""
				} else {
					fmt.Println(err)
					os.Exit(10)
				}
			}
			wasDisplaying = true
		} else if wasDisplaying {
			wasDisplaying = false
			exe := exec.Command(exePath, "display-message", "--text=\"\"")
			_, err := exe.Output()
			if err != nil {
				fmt.Println(err)
			}
		}

	}
}
