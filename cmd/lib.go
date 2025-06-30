package cmd

import (
	"database/sql"
	"github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func MysqlExec(database *sql.DB, query string) (sql.Result, error) {
	Result, err := database.Exec(query)
	if err != nil {
		return nil, err
	}
	return Result, err
}

func MysqlInitConfig(user, pass, address, port, dbname string) *mysql.Config {
	cfg := mysql.Config{
		User:      user,
		Passwd:    pass,
		Net:       "tcp",
		Addr:      strings.Join([]string{address, port}, ":"),
		DBName:    dbname,
		ParseTime: true,
	}
	return &cfg
}

func MysqlPing(cfg *mysql.Config) (*sql.DB, error) {
	var db *sql.DB

	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, err
}

func InitLogger(filePath string, logger *log.Logger, level log.Level) error {
	DirCreate(filepath.Dir(filePath), 0775)
	FileCreate(filePath)
	var err error
	logf, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		//fmt.Println(err)
		return err
	}

	logger.SetOutput(logf)

	if level < log.PanicLevel && level > log.TraceLevel {
		logger.SetLevel(log.InfoLevel)
	} else {
		logger.SetLevel(level)
	}

	logger.SetReportCaller(true)
	logger.SetFormatter(&log.JSONFormatter{PrettyPrint: false})

	return err
}

func DirCreate(dirPath string, permission fs.FileMode) error {
	dirFullPath := dirPath
	err := os.MkdirAll(dirFullPath, permission)
	if err != nil {
		//fmt.Println("Error creating the directory:", err)
		return err
	}
	return err
}

func FileCreate(fullPath string) error {
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		file, err := os.Create(fullPath)
		if err != nil {
			return err
		}
		return file.Close()
	} else {
		return err
	}
}

func AppendLine(filePath, line string) error {
	f, err := os.OpenFile(filePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(line + "\n"); err != nil {
		return err
	}
	return nil
}
