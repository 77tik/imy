# GORM Gen FieldModify 执行过程示例

## 数据库表结构示例

假设我们有一个用户表 `users`，包含以下字段：

```sql
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    profile JSON,
    settings JSON NOT NULL,
    deleted_at TIMESTAMP NULL,
    version INT UNSIGNED DEFAULT 0
);
```

## GORM Gen 执行流程详解

### 第1阶段：数据库结构分析

GORM Gen 连接数据库，查询 `INFORMATION_SCHEMA` 获取表结构：

```go
// 模拟 GORM Gen 内部获取的列信息
columnInfos := []ColumnInfo{
    {
        Name: "id",
        Type: "bigint unsigned",
        Nullable: false,
        IsPrimaryKey: true,
    },
    {
        Name: "name", 
        Type: "varchar(100)",
        Nullable: false,
    },
    {
        Name: "profile",
        Type: "json",
        Nullable: true,  // 可空
    },
    {
        Name: "settings",
        Type: "json", 
        Nullable: false, // 非空
    },
    {
        Name: "deleted_at",
        Type: "timestamp",
        Nullable: true,
    },
    {
        Name: "version",
        Type: "int unsigned",
        Nullable: false,
    },
}
```

### 第2阶段：字段对象构建

GORM Gen 根据数据库信息构建 `gen.Field` 对象：

```go
// 模拟 GORM Gen 内部构建的字段对象
fields := []gen.Field{
    {
        Name: "ID",
        Type: "uint64",
        ColumnName: "id",
        GORMTag: field.GormTag{
            "column": []string{"id"},
            "type": []string{"bigint unsigned"},
            "primaryKey": []string{},
        },
    },
    {
        Name: "Name",
        Type: "string", 
        ColumnName: "name",
        GORMTag: field.GormTag{
            "column": []string{"name"},
            "type": []string{"varchar(100)"},
            "not null": []string{},
        },
    },
    {
        Name: "Profile",
        Type: "string",  // 初始类型
        ColumnName: "profile",
        GORMTag: field.GormTag{
            "column": []string{"profile"},
            "type": []string{"json"},  // 关键：包含 json 类型信息
        },
    },
    {
        Name: "Settings",
        Type: "string",  // 初始类型
        ColumnName: "settings", 
        GORMTag: field.GormTag{
            "column": []string{"settings"},
            "type": []string{"json"},     // 关键：包含 json 类型信息
            "not null": []string{},       // 关键：包含非空约束信息
        },
    },
    {
        Name: "DeletedAt",
        Type: "*time.Time",
        ColumnName: "deleted_at",
        GORMTag: field.GormTag{
            "column": []string{"deleted_at"},
            "type": []string{"timestamp"},
        },
    },
    {
        Name: "Version",
        Type: "uint32",
        ColumnName: "version",
        GORMTag: field.GormTag{
            "column": []string{"version"},
            "type": []string{"int unsigned"},
        },
    },
}
```

### 第3阶段：FieldModify 函数执行

现在执行 `DefaultModelOpt` 中的 `FieldModify` 函数：

```go
// 第一个 FieldModify：处理 JSON 字段
gen.FieldModify(func(f gen.Field) gen.Field {
    // 检查字段的 GORMTag 中是否包含 "json" 类型
    if f.GORMTag["type"][0] == "json" {
        // 对 Profile 字段的处理（可空 JSON）
        if f.ColumnName == "profile" {
            // 添加序列化器标签
            f.GORMTag.Set("serializer", "json")
            // 因为没有 "not null" 约束，生成指针类型
            f.Type = "*Profile"  // SnakeToPascalCase("profile") = "Profile"
            
            // 修改后的字段：
            // Name: "Profile"
            // Type: "*Profile"  // 从 "string" 改为 "*Profile"
            // GORMTag: {
            //     "column": ["profile"],
            //     "type": ["json"],
            //     "serializer": ["json"],  // 新增
            // }
        }
        
        // 对 Settings 字段的处理（非空 JSON）
        if f.ColumnName == "settings" {
            // 添加序列化器标签
            f.GORMTag.Set("serializer", "json")
            // 因为有 "not null" 约束，生成非指针类型
            f.Type = "Settings"  // SnakeToPascalCase("settings") = "Settings"
            
            // 修改后的字段：
            // Name: "Settings"
            // Type: "Settings"  // 从 "string" 改为 "Settings"
            // GORMTag: {
            //     "column": ["settings"],
            //     "type": ["json"],
            //     "not null": [],
            //     "serializer": ["json"],  // 新增
            // }
        }
    }
    return f
})

// 第二个 FieldModify：处理软删除字段
gen.FieldModify(func(f gen.Field) gen.Field {
    if f.ColumnName == "deleted_at" {
        // 添加软删除标志
        f.GORMTag.Set("softDelete", "flag")
        
        // 修改后的字段：
        // Name: "DeletedAt"
        // Type: "*time.Time"
        // GORMTag: {
        //     "column": ["deleted_at"],
        //     "type": ["timestamp"],
        //     "softDelete": ["flag"],  // 新增
        // }
    }
    return f
})
```

### 第4阶段：代码生成

基于修改后的字段信息，GORM Gen 生成最终的 Go 结构体：

```go
// 生成的用户模型结构体
type User struct {
    ID        uint64             `gorm:"column:id;type:bigint unsigned;primaryKey"`
    Name      string             `gorm:"column:name;type:varchar(100);not null"`
    Profile   *Profile           `gorm:"column:profile;type:json;serializer:json"`     // JSON 字段，可空
    Settings  Settings           `gorm:"column:settings;type:json;not null;serializer:json"` // JSON 字段，非空
    DeletedAt *time.Time         `gorm:"column:deleted_at;type:timestamp;softDelete:flag"`   // 软删除字段
    Version   optimisticlock.Version `gorm:"column:version;type:int unsigned"`           // 乐观锁字段
}

// 自动生成的 JSON 结构体类型
type Profile struct {
    Avatar   string `json:"avatar,omitempty"`
    Bio      string `json:"bio,omitempty"`
    Location string `json:"location,omitempty"`
}

type Settings struct {
    Theme        string `json:"theme"`
    Language     string `json:"language"`
    Notification bool   `json:"notification"`
}
```

## 关键时机说明

1. **GORMTag 信息可用性**：在 `FieldModify` 执行时，`f.GORMTag["type"][0]` 已经包含了数据库列的完整类型信息

2. **修改时机**：这是在代码生成**之前**的预处理阶段，不是对已生成代码的后处理

3. **信息来源**：`GORMTag` 中的信息直接来自数据库的 `INFORMATION_SCHEMA`，包括列类型、约束等

4. **修改范围**：可以修改字段的类型、标签、名称等任何属性，这些修改会直接影响最终生成的 Go 代码

## 优势体现

通过这个过程，`dbgen` 实现了：
- **智能类型转换**：JSON 字段自动转换为结构体类型
- **空值处理**：根据数据库约束自动决定是否使用指针类型
- **标签增强**：自动添加序列化器标签和软删除标志
- **零配置**：开发者无需手动配置，完全自动化处理