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

var rootCmd = &cobra.Command{
	Use:   "cloudstackSetupHost",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

	Run: rootRun,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	utils.InitLogger(LogFile, Logger, LogLevel)
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cloudstackSetupHost.toml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			Logger.Error(err)
			os.Exit(1)
		}

		cfgFile = cfgFileDefault
		viper.AddConfigPath(home)
		viper.AddConfigPath("./")
		viper.SetConfigType("toml")
		viper.SetConfigName(cfgFile)
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Errorf("%s file at ./ folder is not exist. Create it first.", cfgFileDefault)
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
	runInterval := viper.GetInt("runInterval")
	fixRam := viper.GetInt("hostRam")
	fixCpus := viper.GetInt("hostCpus")

	var mydb *sql.DB
	var err error
	mysqlCfg := sqlutils.MysqlInitConfig(mysqlUser, mysqlPassword, mysqlHost, "3306", "cloud")
	for {
		mydb, err = sqlutils.MysqlPing(mysqlCfg)
		if err != nil {
			Logger.Errorf("can't connect to Mysqldb at %s: %v, retry after 5 seconds...", err, mysqlHost)
			err := mydb.Close()
			if err != nil {
				Logger.Errorf("error when close mysql connection: %v", err)
			}
			time.Sleep(5 * time.Second)
		} else {
			// keep connection if success
			break
		}
	}

	_, err = MysqlExec(mydb, "update configuration set value = 'false'  where name = 'ca.plugin.root.auth.strictness';")
	if err != nil {
		Logger.Error("Failed update ca.plugin.root.auth.strictness to false, error: ", err)
	}

	job := func(sched *sched.Job) {
		mysqlCfg := sqlutils.MysqlInitConfig(mysqlUser, mysqlPassword, mysqlHost, "3306", "cloud")
		mysqlconnection, err := sqlutils.MysqlPing(mysqlCfg)
		if err != nil {
			Logger.Errorf("can't connect to Mysqldb at %s: %v", err, mysqlHost)
		} else {
			var guids []string
			rows, err := mysqlconnection.Query("SELECT guid FROM host")
			if err != nil {
				Logger.Error(err)
			}
			defer rows.Close()
			for rows.Next() {
				var uuid string
				if err := rows.Scan(&uuid); err != nil {
					Logger.Error(err)
				}
				guids = append(guids, uuid)
			}
			if err := rows.Err(); err != nil {
				log.Fatal(err)
			}

			for _, guid := range guids {
				/* 				newUuid, _ := uuid.NewRandom()
				   				newUuidString := newUuid.String()
				   				host_uuid := "cloudstack" + newUuidString[8:]
				   				host_guid := host_uuid + "-LibvirtComputingResource"
				   				//fmt.Println(v)
				   				if !strings.Contains(guid, "cloudstack") && strings.Contains(guid, "LibvirtComputingResource") {
				   					//queryRam := fmt.Sprintf("select ram from host where guid = '%s';", guid)
				   					//queryCpu := fmt.Sprintf("select cpus from host where guid = '%s';", guid)
				   					//var realCpus, realRam int
				   					//mysqlconnection.QueryRow(queryCpu).Scan(&realCpus)
				   					//mysqlconnection.QueryRow(queryRam).Scan(&realRam)
				   					//fmt.Println(realCpus, realRam)
				   					updateUUIDquery := fmt.Sprintf("update host set uuid = '%s', guid = '%s' where guid = '%s';", host_uuid, host_guid, guid)
				   					_, err := MysqlExec(mysqlconnection, updateUUIDquery)
				   					if err != nil {
				   						Logger.Errorf("failed update Uuid using command: %s, Error: %v", updateUUIDquery, err)
				   					}
				   				}
				*/
				updateRamCpus := fmt.Sprintf("update host set ram = %d, cpus = %d where guid = '%s';", fixRam, fixCpus, guid)
				_, err := MysqlExec(mysqlconnection, updateRamCpus)
				if err != nil {
					Logger.Errorf("failed update Uuid using command: %s, Error: %v", updateRamCpus, err)
				}
			}
		}
		defer mysqlconnection.Close()
	}
	sched.Every(runInterval).ESeconds().Run(job)

	defer mydb.Close()
	// Keep the program from not exiting.
	runtime.Goexit()
}

func MysqlExec(database *sql.DB, query string) (sql.Result, error) {
	Result, err := database.Exec(query)
	if err != nil {
		return nil, err
	}
	return Result, err
}
