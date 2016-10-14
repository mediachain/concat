#!/bin/bash

gx-go rewrite && go install -tags=embed ./... && gx-go rewrite --undo
