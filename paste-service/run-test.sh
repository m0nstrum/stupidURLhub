#!/bin/bash

# Запуск paste-service в тестовом режиме без подключения к базе данных
export TEST_MODE=true
export CACHE_TYPE=inmemory
export CACHE_DEFAULTTTL=30m
export CACHE_REFRESHTTLONGET=true
go run main.go 