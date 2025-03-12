package db

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

var Dao *gorm.DB

func Init(sqlitePath string) {
	dbLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second * 3,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      false,
			LogLevel:                  logger.Info,
		},
	)
	var openDb *gorm.DB
	var err error
	if sqlitePath == "" {
		sqlitePath = "data/stock.db?cache=shared&mode=rwc&_journal_mode=WAL&_cache_size=-2000&page_size=4096"
	}

	openDb, err = gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		Logger:                                   dbLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              true,
	})
	//读写分离提高sqlite效率，防止锁库
	openDb.Use(dbresolver.Register(
		dbresolver.Config{
			Replicas: []gorm.Dialector{sqlite.Open(sqlitePath)}},
	))

	if err != nil {
		log.Fatalf("db connection error is %s", err.Error())
	}

	dbCon, err := openDb.DB()
	if err != nil {
		log.Fatalf("openDb.DB error is  %s", err.Error())
	}
	dbCon.SetMaxIdleConns(10)
	dbCon.SetMaxOpenConns(100)
	dbCon.SetConnMaxLifetime(time.Hour)
	Dao = openDb
}

func InitDB() *gorm.DB {
	// 修改缓存大小设置
	cacheSize := -2000 // 使用负值表示以KB为单位，-2000表示2MB
	db, err := gorm.Open(sqlite.Open("data.db?cache=shared&_journal_mode=WAL&_cache_size="+strconv.Itoa(cacheSize)+"&page_size=4096"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	//读写分离配置也需要使用相同的参数
	db.Use(dbresolver.Register(
		dbresolver.Config{
			Replicas: []gorm.Dialector{
				sqlite.Open("data.db?cache=shared&_journal_mode=WAL&_cache_size=" + strconv.Itoa(cacheSize) + "&page_size=4096"),
			},
		},
	))

	if err != nil {
		log.Fatalf("db connection error is %s", err.Error())
	}

	dbCon, err := db.DB()
	if err != nil {
		log.Fatalf("openDb.DB error is  %s", err.Error())
	}
	dbCon.SetMaxIdleConns(10)
	dbCon.SetMaxOpenConns(100)
	dbCon.SetConnMaxLifetime(time.Hour)
	return db
}
