#!/bin/sh
exec nc {{ .ClusterIP }} {{ .Port }}
