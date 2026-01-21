package config

import (
	"path/filepath"
	"strings"

	"github.com/gotify/server/v2/mode"
	"github.com/jinzhu/configor"
)

type Configuration struct {
	Server struct {
		KeepAlivePeriodSeconds int
		ListenAddr             string `default:""`
		Port                   int    `default:"80"`

		SSL struct {
			Enabled         bool   `default:"false"`
			RedirectToHTTPS bool   `default:"true"`
			ListenAddr      string `default:""`
			Port            int    `default:"443"`
			CertFile        string `default:""`
			CertKey         string `default:""`
			LetsEncrypt     struct {
				Enabled      bool   `default:"false"`
				AcceptTOS    bool   `default:"false"`
				Cache        string `default:"data/certs"`
				DirectoryURL string `default:""`
				Hosts        []string
			}
		}
		ResponseHeaders map[string]string
		Stream          struct {
			PingPeriodSeconds int `default:"45"`
			AllowedOrigins    []string
		}
		Cors struct {
			AllowOrigins []string
			AllowMethods []string
			AllowHeaders []string
		}

		TrustedProxies []string
		RateLimit struct {
			Global struct {
				Enabled           bool `default:"true"`
				RequestsPerSecond int  `default:"20"`
				Burst             int  `default:"50"`
			}
			Auth struct {
				Enabled           bool `default:"true"`
				RequestsPerSecond int  `default:"10"`
				Burst             int  `default:"20"`
			}
			Message struct {
				Enabled           bool `default:"true"`
				RequestsPerSecond int  `default:"15"`
				Burst             int  `default:"30"`
			}
			Admin struct {
				Enabled           bool `default:"true"`
				RequestsPerSecond int  `default:"5"`
				Burst             int  `default:"10"`
			}
		}
		AuthBlacklist struct {
			Enabled         bool     `default:"true"`
			MaxFailures     int      `default:"5"`
			WindowSeconds   int      `default:"300"`
			BlockDuration   int      `default:"3600"`
			Whitelist       []string
			CleanupInterval int      `default:"300"`
		}
	}
	Database struct {
		Dialect    string `default:"sqlite3"`
		Connection string `default:"data/gotify.db"`
	}
	DefaultUser struct {
		Name string `default:"admin"`
		Pass string `default:"admin"`
	}
	PassStrength      int    `default:"10"`
	UploadedImagesDir string `default:"data/images"`
	PluginsDir        string `default:"data/plugins"`
	Registration      bool   `default:"false"`
}

func configFiles() []string {
	if mode.Get() == mode.TestDev {
		return []string{"config.yml"}
	}
	return []string{"config.yml", "/etc/gotify/config.yml"}
}

func Get() *Configuration {
	conf := new(Configuration)
	err := configor.New(&configor.Config{ENVPrefix: "GOTIFY", Silent: true}).Load(conf, configFiles()...)
	if err != nil {
		panic(err)
	}
	addTrailingSlashToPaths(conf)
	return conf
}

func addTrailingSlashToPaths(conf *Configuration) {
	if !strings.HasSuffix(conf.UploadedImagesDir, "/") && !strings.HasSuffix(conf.UploadedImagesDir, "\\") {
		conf.UploadedImagesDir += string(filepath.Separator)
	}
}
