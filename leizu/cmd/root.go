package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "leizu",
	Short: "A envoy api implemention for kubernetes",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().BoolP("out-cluster", "o", false, "using out cluster kubeconfig")
	viper.BindPFlag("OutCluster", rootCmd.Flags().Lookup("out-cluster"))

	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defaultKubeConfig := filepath.Join(home, ".kube", "config")
	rootCmd.Flags().StringP("kubeconfig", "k", defaultKubeConfig, "absolute path to the kubeconfig file")
	viper.BindPFlag("KubeConfigPath", rootCmd.Flags().Lookup("kubeconfig"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".github.com\shanbay\leizu" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".leizu.yml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
