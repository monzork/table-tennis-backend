#!/bin/bash
TOKEN=$(curl -s -X POST http://localhost:8080/admin/login -d "username=admin&password=tabletennis2024" -c cookies.txt)
curl -i -X POST http://localhost:8080/tournaments -b cookies.txt -d "name=Test2&type=singles&format=elimination&startDate=2026-04-03&endDate=2026-04-04&groupPassCount=2"
