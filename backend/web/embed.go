// Package web 承载 H5 构建产物：交付时由 make release 复制 frontend/dist/build/h5 到 web/dist，
// 再随二进制一起打包，浏览器打开 http://localhost:8080 即可使用。
package web

import "embed"

//go:embed all:dist
var Files embed.FS
