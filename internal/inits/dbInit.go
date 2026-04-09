package inits

import (
	"database/sql"
	"log"
	"os"

	"github/JustGopher/Gotaxy/internal/storage/models"

	_ "modernc.org/sqlite"
)

// DBInit 数据库初始化
func DBInit(errorLog *log.Logger) *sql.DB {
	var err error
	// 检查并创建 data 文件夹
	dataDir := "data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err := os.MkdirAll(dataDir, os.ModePerm)
		if err != nil {
			if errorLog != nil {
				errorLog.Printf("DBInit() 创建 data 文件夹失败 -> %v", err)
			}
			panic(err)
		}
	}

	db, err := sql.Open("sqlite", "data/data.db")
	if err != nil {
		if errorLog != nil {
			errorLog.Printf("DBInit() 打开数据库失败 -> %v", err)
		}
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		if errorLog != nil {
			errorLog.Printf("DBInit() 数据库连接失败 -> %v", err)
		}
		panic(err)
	}

	err = models.CreateCfgStructure(db)
	if err != nil {
		if errorLog != nil {
			errorLog.Printf("DBInit() 创建配置表结构失败 -> %v", err)
		}
		panic(err)
	}
	err = models.CreateMpgStructure(db)
	if err != nil {
		if errorLog != nil {
			errorLog.Printf("DBInit() 创建映射表结构失败 -> %v", err)
		}
		panic(err)
	}
	return db
}
