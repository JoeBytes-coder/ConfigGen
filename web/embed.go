package web

import "embed"

//go:embed index.html views/*
var FS embed.FS
