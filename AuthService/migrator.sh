#!/bin/bash
source .env
sleep 5 && migrate -path "./migrations/" -database "${MIGR_DSN}" up
