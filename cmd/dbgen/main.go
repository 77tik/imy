package main

import (
	"fmt"
	"path"
	"path/filepath"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"imy/pkg/dbgen"
)

var MysqlDsn = "root:root123@tcp(127.0.0.1:3306)/imydb?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"

func Mysql() {
	g := gen.NewGenerator(gen.Config{
		OutPath:          filepath.Join(".", "internal", "dao"),
		OutFile:          "dao.go",
		ModelPkgPath:     filepath.Join(".", "internal", "dao", "model"),
		Mode:             gen.WithDefaultQuery, // generate mode
		FieldWithTypeTag: true,
	})

	g.WithImportPkgPath("gorm.io/gorm", "gorm.io/plugin/optimisticlock", "gorm.io/plugin/soft_delete")

	db, _ := gorm.Open(mysql.Open(MysqlDsn))

	g.UseDB(db)

	g.WithDataTypeMap(dbgen.DataMapMySQL)

	for _, opt := range dbgen.DefaultModelOpt {
		g.WithOpts(opt)
	}

	g.ApplyBasic(
		g.GenerateAllTable()...,
	)

	g.Execute()

	// 生成其他代码
	tables, _ := db.Migrator().GetTables()

	for _, table := range tables {
		modelStructName := db.Config.NamingStrategy.SchemaName(table)

		columnTypes, err := db.Migrator().ColumnTypes(table)
		if err != nil {
			fmt.Println(err)
			return
		}

		dbParams, err := dbgen.Params(table, modelStructName, columnTypes, dbgen.DataMapMySQL)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = dbgen.Build(dbParams, path.Join("./internal/dao/", table+".dao.go"))
		if err != nil {
			fmt.Println(err)
			return
		}

		err = dbgen.BuildCustom(dbParams, path.Join("./internal/dao/", table+".go"))
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func main() {
	Mysql()
}
