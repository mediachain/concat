#!/bin/bash

gx-go rewrite && go build ./... && gx-go rewrite --undo
