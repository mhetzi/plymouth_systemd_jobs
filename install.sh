#!/bin/bash
sudo dnf install golang

go build .
sudo systemctl stop plymouth-boot-jobs.service plymouth-boot-jobs-poweroff.service
sudo cp plymouth_systemd_jobs /usr/bin
sudo cp plymouth-boot-jobs.service /lib/systemd/system/
sudo cp plymouth-boot-jobs-poweroff.service /lib/systemd/system/
sudo cp -n plymouth_systemd_job.toml /etc/default/
sudo systemctl enable plymouth-boot-jobs.service
sudo systemctl enable plymouth-boot-jobs-poweroff.service
