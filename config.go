package leizu

import "github.com/spf13/viper"

// all config fields
func loadDefaultSettingsFor(v *viper.Viper){
	v.SetDefault("OutCluster", false)
	v.SetDefault("KubeConfigPath", "")

}