#!/bin/bash

# Запуск paste-service в тестовом режиме без подключения к базе данных
export TEST_MODE=true
go run main.go 