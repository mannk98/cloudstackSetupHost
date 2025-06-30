/*
Copyright © 2024 mannk khacman98@gmail.com
*/
package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/sonnt85/gosutils/sched"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile          string
	Logger           = log.New()
	LogLevel         = log.ErrorLevel
	LogFile          = "cloudstackSetupHost.log"
	cfgFileDefault   = ".cloudstackSetupHost"
	AlreadySetupFile = ".cloudstackSetupHost_HostAlreadySetup"
)

var rootCmd = &cobra.Command{
	Use:   "cloudstackSetupHost",
	Short: "Cloudstack Setup Host",
	Long:  ``,

	Run: rootRun,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	InitLogger(LogFile, Logger, LogLevel)
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
		currentDir, _ := os.Getwd()
		viper.AddConfigPath(currentDir)
		viper.SetConfigType("toml")
		viper.SetConfigName(cfgFile)
	}

	mysqlUser := os.Getenv("CLOUDSTACK_MYSQL_USER")
	mysqlPassword := os.Getenv("CLOUDSTACK_MYSQL_PASSWORD")
	viper.SetDefault("mysqlHost", "localhost")
	viper.SetDefault("runInterval", 60)
	viper.Set("mysqlUser", mysqlUser)
	viper.Set("mysqlPassword", mysqlPassword)

	errWriteCfg := viper.WriteConfigAs(cfgFile + ".toml")
	if errWriteCfg != nil {
		Logger.Errorf("Error writing config: %v", errWriteCfg)
		os.Exit(1)
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
	errCreateFile := FileCreate(AlreadySetupFile)
	if errCreateFile != nil {
		if os.IsExist(errCreateFile) {
			Logger.Info("File ~/.CloudstackSetupHost_HostAlreadySetup already exists, exiting...")
		} else {
			Logger.Error("Error creating file: ", errCreateFile)
			os.Exit(1)
		}
	}

	mysqlUser := viper.GetString("mysqluser")
	mysqlPassword := viper.GetString("mysqlpassword")
	mysqlHost := viper.GetString("mysqlhost")
	runInterval := viper.GetInt("runinterval")

	var mydb *sql.DB
	var err error
	mysqlCfg := MysqlInitConfig(mysqlUser, mysqlPassword, mysqlHost, "3306", "cloud")
	for {
		mydb, err = MysqlPing(mysqlCfg)
		if err != nil {
			Logger.Errorf("can't connect to Mysqldb at %s: %v, retry after 10 seconds...", err, mysqlHost)
			defer mydb.Close()
			time.Sleep(10 * time.Second)
		} else {
			// keep connection if success
			break
		}
	}

	_, err = MysqlExec(mydb, "UPDATE configuration set value = 'false'  WHERE name = 'ca.plugin.root.auth.strictness';")
	if err != nil {
		Logger.Error("Failed update ca.plugin.root.auth.strictness to false, error: ", err)
	}

	job := func(sched *sched.Job) {

		if err != nil {
			Logger.Errorf("can't connect to Mysqldb at %s: %v", err, mysqlHost)
		} else {
			var guids []string
			rows, errQuery := mydb.Query("SELECT guid FROM host")
			if errQuery != nil {
				Logger.Error(errQuery)
			}
			defer rows.Close()
			for rows.Next() {
				var uuid sql.NullString
				if errScan := rows.Scan(&uuid); errScan != nil {
					Logger.Errorf("Error scanning row: %v", errScan)
					continue
				}
				// Skip if UUID is NULL
				if !uuid.Valid {
					Logger.Debug("Skipping NULL UUID")
					continue
				}
				// find Hosts only
				if strings.Contains(uuid.String, "StorageResource") || strings.Contains(uuid.String, "ProxyResource") {
					continue
				} else if strings.Contains(uuid.String, "LibvirtComputingResource") {
					guids = append(guids, uuid.String)
				}
			}
			if err := rows.Err(); err != nil {
				log.Fatal(err)
			}

			listSetupHost, errRead := os.ReadFile(AlreadySetupFile)
			if errRead != nil {
				Logger.Errorf("Error reading file %s: %v", AlreadySetupFile, errRead)
				os.Exit(1)
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
				if !strings.Contains(string(listSetupHost), guid) {
					getCurrentRam := fmt.Sprintf("SELECT ram FROM host WHERE guid = '%s';", guid)
					getCurrentCpus := fmt.Sprintf("SELECT cpus FROM host WHERE guid = '%s';", guid)
					var currentRam, currentCpus int
					errRam := mydb.QueryRow(getCurrentRam).Scan(&currentRam)
					if errRam != nil {
						Logger.Errorf("Failed to get current RAM for host %s using command: %s, Error: %v", guid, getCurrentRam, errRam)
						continue
					}
					errCpus := mydb.QueryRow(getCurrentCpus).Scan(&currentCpus)
					if errCpus != nil {
						Logger.Errorf("Failed to get current CPU for host %s using command: %s, Error: %v", guid, getCurrentCpus, errCpus)
						continue
					}
					Logger.Infof("Current RAM for host %s is %d, current CPU is %d", guid, currentRam, currentCpus)

					updateRam := currentRam * 2
					updateCpus := currentCpus * 3
					updateRamCpus := fmt.Sprintf("UPDATE host set ram = %d, cpus = %d WHERE guid = '%s';", updateRam, updateCpus, guid)
					_, errExec := MysqlExec(mydb, updateRamCpus)
					if errExec != nil {
						Logger.Errorf("Failed to update RAM and CPU for host %s using command: %s, Error: %v", guid, updateRamCpus, errExec)
						continue
					} else {
						Logger.Infof("Host %s has been updated with fixed RAM and CPU.", guid)
					}

					errWrite := AppendLine(AlreadySetupFile, guid)
					if errWrite != nil {
						Logger.Errorf("Error writing to file %s: %v", AlreadySetupFile, errWrite)
					} else {
						Logger.Infof("Host %s has been setup with fixed RAM and CPU.", guid)
					}
				}
			}
		}
	}

	sched.Every(runInterval).ESeconds().Run(job)

	defer mydb.Close()
	// Keep the program from not exiting.
	runtime.Goexit()
}
