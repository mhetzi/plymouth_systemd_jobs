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

func checkCmdArgs(settings *SettingsStruct) bool {
	cmdArgs := os.Args[1:]
	if len(cmdArgs) > 0 {
		if cmdArgs[0] == "show_overrides" {
			loadSettings(true)
			fmt.Printf("All override:\n %+v \n", settings.messages)
			return true
		} else if cmdArgs[0] == "help" {
			fmt.Println("==== HELP ====")
			fmt.Println("help: Shows this Text")
			fmt.Println("show_overrides: Shows all defined custom service Texts")
			return true
		}
	}
	return false
}

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

	settings, err := loadSettings(false)
	if err != nil {
		fmt.Println(err)
	}

	if checkCmdArgs(settings) {
		os.Exit(0)
	}

	go processJobs(settings)

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
	settings = nil
}

func processJobs(settings *SettingsStruct) {
	fmt.Println("Start timeloop...")

	wasDisplaying := false
	var exePath string = ""

	for range time.Tick(time.Second * 1) {
		if exePath == "" {
			exePath = which.Which("plymouth")
			fmt.Printf("ExecPath=%s\n", exePath)
			continue
		}
		current_jobs, err := getOldestJob()
		if err != nil && current_jobs.watch.ElapsedSeconds() > float64(settings.condis.min_time_secs) {
			fmt.Println(current_jobs)
			txtCustom := settings.getCustomMessage(&current_jobs)
			txtArg := fmt.Sprintf("--text=\"[Time:%.f] Job: Unit: %s %s\"", current_jobs.watch.ElapsedSeconds(), current_jobs.sUnit, txtCustom)
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
