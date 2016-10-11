#!/bin/bash

gx-go rewrite && go build -tags=embed ./... && gx-go rewrite --undo
