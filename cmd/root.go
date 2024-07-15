/*
Copyright © 2024 mannk khacman98@gmail.com
*/
package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/mannk98/goutils/sqlutils"
	"github.com/mannk98/goutils/utils"
	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sched"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile        string
	Logger         = log.New()
	LogLevel       = log.ErrorLevel
	LogFile        = "cloudstackSetupHost.log"
	cfgFileDefault = ".cloudstackSetupHost"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cloudstackSetupHost",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: rootRun,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	utils.InitLogger(LogFile, Logger, LogLevel)
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cloudstackSetupHost.toml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			Logger.Error(err)
			os.Exit(1)
		}

		cfgFile = cfgFileDefault
		// Search config in home directory with name ".cloudstackSetupHost" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath("./")
		viper.SetConfigType("toml")
		viper.SetConfigName(cfgFile)
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Error("config.toml file at ./ folder is not exist. Create it first.")
		} else {
			Logger.Error(err)
		}
	} else {
		Logger.Info("Using config file:", viper.ConfigFileUsed())
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	mysqlUser := viper.GetString("mysqlUser")
	mysqlPassword := viper.GetString("mysqlPassword")
	mysqlHost := viper.GetString("mysqlHost")
	//defaultHostName := viper.GetString("defaultHostName")
	runInterval := viper.GetInt("runInterval")

	var mysqlconnection *sql.DB
	var err error
	for {
		mysqlCfg := sqlutils.MysqlInitConfig(mysqlUser, mysqlPassword, mysqlHost, "3306", "cloud")
		mysqlconnection, err = sqlutils.MysqlPing(mysqlCfg)
		if err != nil {
			Logger.Errorf("can't connect to Mysqldb at %s: %v", err, mysqlHost)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	_, err = MysqlExec(mysqlconnection, "update configuration set value = 'false'  where name = 'ca.plugin.root.auth.strictness';")
	if err != nil {
		Logger.Error("Failed update ca.plugin.root.auth.strictness to false, error: ", err)
	}

	job := func(sched *sched.Job) {
		mysqlCfg := sqlutils.MysqlInitConfig(mysqlUser, mysqlPassword, mysqlHost, "3306", "cloud")
		mysqlconnection, err := sqlutils.MysqlPing(mysqlCfg)
		if err != nil {
			Logger.Errorf("can't connect to Mysqldb at %s: %v", err, mysqlHost)
		} else {
			var ListHostUUID []string
			// set new variable
			/* 			newUuid, _ := uuid.NewRandom()
			   			newUuidString := newUuid.String()
			   			host_uuid := "cloudstack" + newUuidString[8:]
			   			host_guid := host_uuid + "-LibvirtComputingResource" */

			mysqlconnection.QueryRow("select uuid from host").Scan(&ListHostUUID)
			for _, v := range ListHostUUID {
				fmt.Println(v)
			}

			/* 			queryRam := fmt.Sprintf("select ram from host where name = '%s';", defaultHostName)
			   			queryCpu := fmt.Sprintf("select cpus from host where name = '%s';", defaultHostName)

			   			var realCpus, realRam int
			   			mysqlconnection.QueryRow(queryCpu).Scan(&realCpus)
			   			mysqlconnection.QueryRow(queryRam).Scan(&realRam)

			   			//updateHostName := fmt.Sprintf(" update host set name = 'newHost-%s' where uuid = '%s';", host_uuid.String(), host_uuid.String())

			   			updateUUID := fmt.Sprintf("update host set uuid = '%s', guid = '%s', ram = %d, cpus = %d where name = 'cloudstack';", host_uuid, host_guid, realRam*3, realCpus*3)
			   			_, err := MysqlExec(mysqlconnection, updateUUID)
			   			if err != nil {
			   				Logger.Errorf("failed update Uuid using command: %s, Error: %v", updateUUID, err)
			   			} */
			/* 				_, err = MysqlExec(mysqlconnection, updateHostName)
			   				if err != nil {
			   					Logger.Errorf("failed update Uuid using command: %s, Error: %v", updateHostName, err)
			   				} */

		}

	}
	sched.Every(runInterval).ESeconds().Run(job)
	// Keep the program from not exiting.
	defer mysqlconnection.Close()
	runtime.Goexit()
}

func MysqlExec(database *sql.DB, query string) (sql.Result, error) {
	//doc_type := 0
	//query := fmt.Sprintf("UPDATE history SET request_datetime = DATE_ADD(request_datetime, INTERVAL 7 HOUR) WHERE doc_type=%d AND request_datetime BETWEEN '2024-02-01 00:00:00' AND '2024-02-20 23:59:59'", doc_type)

	//fmt.Println(query)
	Result, err := database.Exec(query)
	if err != nil {
		return nil, err
	}
	return Result, err
}
