package dbgen

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func TypeNullable(columnType gorm.ColumnType, dataType string) string {
	if n, ok := columnType.Nullable(); ok && n {
		return fmt.Sprintf("*%s", dataType)
	} else {
		return dataType
	}
}

func getCandStr(colGo, colGoType string) string {
	switch colGoType {
	case "bool":
		return "true"
	case "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64", "float32", "float64":
		return fmt.Sprintf(`params.%s != 0`, colGo)
	case "string":
		return fmt.Sprintf(`params.%s != ""`, colGo)
	case "[]byte", "[]uint8":
		return fmt.Sprintf(`len(params.%s) > 0`, colGo)
	case "time.Time":
		return fmt.Sprintf(`!params.%s.IsZero()`, colGo)
	default:
		return "true"
	}
}

var DataMapMySQL = map[string]func(gorm.ColumnType) (dataType string){
	"numeric": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int32") },
	"integer": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int32") },
	"tinyint": func(columnType gorm.ColumnType) (dataType string) {
		ct, _ := columnType.ColumnType()
		if strings.HasPrefix(ct, "tinyint(1)") {
			if columnType.Name() == "deleted_at" {
				return "soft_delete.DeletedAt"
			}
			return TypeNullable(columnType, "bool")
		}
		if strings.HasSuffix(ct, "unsigned") {
			return TypeNullable(columnType, "uint8")
		}
		return TypeNullable(columnType, "int8")
	},
	"smallint": func(columnType gorm.ColumnType) (dataType string) {
		ct, _ := columnType.ColumnType()
		if strings.HasSuffix(ct, "unsigned") {
			return TypeNullable(columnType, "uint16")
		}
		return TypeNullable(columnType, "int16")
	},
	"mediumint": func(columnType gorm.ColumnType) (dataType string) {
		ct, _ := columnType.ColumnType()
		if strings.HasSuffix(ct, "unsigned") {
			return TypeNullable(columnType, "uint32")
		}
		return TypeNullable(columnType, "int32")
	},
	"int": func(columnType gorm.ColumnType) (dataType string) {
		ct, _ := columnType.ColumnType()
		if strings.HasSuffix(ct, "unsigned") {
			return TypeNullable(columnType, "uint32")
		}
		return TypeNullable(columnType, "int32")
	},
	"bigint": func(columnType gorm.ColumnType) (dataType string) {
		ct, _ := columnType.ColumnType()
		if strings.HasSuffix(ct, "unsigned") {
			return TypeNullable(columnType, "uint64")
		}
		return TypeNullable(columnType, "int64")
	},
	"float":      func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"real":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"double":     func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"decimal":    func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"char":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"varchar":    func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"tinytext":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"mediumtext": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"longtext":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"binary":     func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]byte") },
	"varbinary":  func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]byte") },
	"tinyblob":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]byte") },
	"blob":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]byte") },
	"mediumblob": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]byte") },
	"longblob":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]byte") },
	"text":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"json":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"enum":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },
	"time":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "time.Time") },
	"date":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "time.Time") },
	"datetime":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "time.Time") },
	"timestamp":  func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "time.Time") },
	"year":       func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int32") },
	"bit":        func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "[]uint8") },
	"boolean":    func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "bool") },
}

var DataMapClickHouse = map[string]func(gorm.ColumnType) (dataType string){
	"Int8":    func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int8") },
	"Int16":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int16") },
	"Int32":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int32") },
	"Int64":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "int64") },
	"UInt8":   func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "uint8") },
	"UInt16":  func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "uint16") },
	"UInt32":  func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "uint32") },
	"UInt64":  func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "uint64") },
	"Float32": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float32") },
	"Float64": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"String":  func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "string") },

	"AggregateFunction(max, Float64)": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"AggregateFunction(min, Float64)": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
	"AggregateFunction(avg, Float64)": func(columnType gorm.ColumnType) (dataType string) { return TypeNullable(columnType, "float64") },
}
