#!/bin/bash

# Запуск paste-service в тестовом режиме без подключения к базе данных
# если не указать время то ттлка будет 0 и кэш не заработает
export SERVER_TESTMODE=true
export CACHE_TYPE=inmemory
export CACHE_DEFAULTTTL=30m
export CACHE_REFRESHTTLONGET=true
go run main.go 