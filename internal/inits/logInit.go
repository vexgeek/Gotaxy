package inits

import (
	"log"
	"path/filepath"

	"github/JustGopher/Gotaxy/pkg/logger"
)

// LogInit 日志初始化
func LogInit() (*log.Logger, *log.Logger) {
	logPath := "./logs"

	// 获取日志输出目标（info文件）
	infoPath := filepath.Join(logPath, "info")
	infoFile := logger.NewRollingFileWriter(infoPath)
	infoWriter := log.New(infoFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)

	// 获取日志输出目标（error文件）
	errorPath := filepath.Join(logPath, "error")
	errorFile := logger.NewRollingFileWriter(errorPath)
	errorWriter := log.New(errorFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

	return infoWriter, errorWriter
}
