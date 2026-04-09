package config

import (
	"database/sql"
	"fmt"
	"strconv"

	"github/JustGopher/Gotaxy/internal/session"
	"github/JustGopher/Gotaxy/internal/storage/models"
)

// Config 配置
type Config struct {
	Name         string `json:"name"`
	ServerIP     string `json:"server_ip"`
	ListenPort   string `json:"listen_port"`
	Email        string `json:"email"`
	TotalTraffic int64  `json:"total_traffic"` // 总流量
}

// ConfigLoad 配置加载
func (cfg *Config) ConfigLoad(db *sql.DB, sessionMgr *session.Manager) {
	cfgMap, err := models.GetAllCfg(db)
	if err != nil {
		fmt.Printf("ConfigLoad() 查询配置数据失败 -> %v", err)
		return
	}
	if len(cfgMap) != 4 {
		err = models.InsertCfg(db, "server_ip", "127.0.0.1")
		if err != nil {
			fmt.Printf("ConfigLoad() 创建配置数据失败 -> %v", err)
		}
		err = models.InsertCfg(db, "listen_port", "9000")
		if err != nil {
			fmt.Printf("ConfigLoad() 创建配置数据失败 -> %v", err)
		}
		err = models.InsertCfg(db, "email", "")
		if err != nil {
			fmt.Printf("ConfigLoad() 创建配置数据失败 -> %v", err)
		}
		err = models.InsertCfg(db, "total_traffic", "0")
		if err != nil {
			fmt.Printf("ConfigLoad() 创建配置数据失败 -> %v", err)
		}
		cfgMap, err = models.GetAllCfg(db)
		if err != nil {
			fmt.Printf("ConfigLoad() 创建配置数据失败 -> %v", err)
			return
		}
	}
	cfg.ServerIP = cfgMap["server_ip"]
	cfg.ListenPort = cfgMap["listen_port"]
	cfg.Email = cfgMap["email"]
	cfg.TotalTraffic, err = strconv.ParseInt(cfgMap["total_traffic"], 10, 64)
	if err != nil {
		fmt.Printf("ConfigLoad() 解析配置数据失败 -> %v", err)
		return
	}

	fmt.Println("已加载配置...")
	mpg, err := models.GetAllMpg(db)
	if err != nil {
		fmt.Printf("ConfigLoad() 查询映射数据失败 -> %v", err)
		return
	}
	for _, v := range mpg {
		if v.Enable {
			sessionMgr.SetMapping(v.Name, v.PublicPort, v.TargetAddr, true, v.Traffic, v.RateLimit)
		} else {
			sessionMgr.SetMapping(v.Name, v.PublicPort, v.TargetAddr, false, v.Traffic, v.RateLimit)
		}
	}
}
