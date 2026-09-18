package web

import "embed"

// FS 持有 Vite 构建产物（web/dist），由 Go 编译期嵌入。
// 重新构建前端：cd web && npm install && npm run build
//
//go:embed all:dist
var FS embed.FS
