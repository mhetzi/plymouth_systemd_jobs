package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

const (
	ETC_SETTINGS_PATH = "/etc/default/plymouth_systemd_job.toml"
)

type conditions struct {
	min_time_secs int64
}

type SettingsStruct struct {
	messages map[string]string
	condis   conditions
}

func loadSettings(debug bool) (*SettingsStruct, error) {
	newSettings := new(SettingsStruct)
	newSettings.messages = make(map[string]string, 2)
	newSettings.condis = conditions{
		min_time_secs: 5,
	}

	if _, err := os.Stat(ETC_SETTINGS_PATH); errors.Is(err, os.ErrNotExist) {
		return newSettings, err
	}

	file, err := os.ReadFile(ETC_SETTINGS_PATH)

	if err != nil {
		return newSettings, err
	}

	temp := make(map[string]interface{})
	err = toml.Unmarshal(file, &temp)
	if err != nil {
		return nil, err
	}

	if debug {
		fmt.Printf("toml v2: %#v\n", temp)
	}
	msgs, ok := temp["messages"]
	if ok {
		for unit, text := range msgs.(map[string]interface{}) {
			if debug {
				fmt.Printf("toml v2: %#v: %#v\n", unit, text)
			}
			newSettings.messages[unit] = text.(string)
		}
	}
	condi_test, ok := temp["condition"]
	if ok {
		condi := condi_test.(map[string]interface{})
		min_secs, ok := condi["min_secs"]
		if ok {
			newSettings.condis.min_time_secs = min_secs.(int64)
		}
	}

	return newSettings, nil
}

func (s *SettingsStruct) getCustomMessage(job *Job) string {
	return s.messages[job.sUnit]
}
