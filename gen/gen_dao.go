package main

import (
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type Querier interface {
	FilterWithNameAndRole(name, role string) ([]gen.T, error)
}

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "./dao",
		Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})

	gormdb, _ := gorm.Open(mysql.Open("root:5023152@(127.0.0.1:3306)/dredger?charset=utf8mb4&parseTime=True&loc=Local"))
	g.UseDB(gormdb)

	g.ApplyBasic(g.GenerateAllTable()...)

	g.Execute()
}
