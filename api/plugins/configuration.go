package plugins

import (
	"github.com/Glintt/gAPI/api/config"
	"path"
)

func Configurations() config.GApiPluginsConfig {
	if config.GApiConfiguration.Plugins.Location == "" {
		config.GApiConfiguration.Plugins.Location = path.Dir(PLUGINS_LOCATION) + "/"
	}

	return config.GApiConfiguration.Plugins
}
