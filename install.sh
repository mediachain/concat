#!/bin/bash

gx-go rewrite && go install ./... && gx-go rewrite --undo
