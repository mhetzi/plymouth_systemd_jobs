#!/bin/bash
sudo dnf install golang

go build .
sudo cp plymouth_systemd_jobs /usr/bin
sudo cp plymouth-boot-jobs.service /lib/systemd/system/
sudo cp plymouth-boot-jobs-poweroff.service /lib/systemd/system/
sudo systemctl enable plymouth-boot-jobs.service
sudo systemctl enable plymouth-boot-jobs-poweroff.service