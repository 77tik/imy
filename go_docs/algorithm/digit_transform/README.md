# 数字位奇偶转换算法

## 问题描述

给定一个数字，将其每一位按照奇偶性进行转换：
- 奇数位变为 1
- 偶数位变为 0

### 示例
- `2222` → `0` (所有位都是偶数2，转换为0000，即0)
- `3333` → `1111` (所有位都是奇数3，转换为1111)
- `1234` → `1010` (1→1, 2→0, 3→1, 4→0)
- `987654321` → `101010101`

## 算法实现

本项目提供了多种实现方式：

### 1. TransformDigits (字符串方法)
```go
func TransformDigits(num int) int
```
- **原理**：将数字转换为字符串，逐位处理后重新组合
- **优点**：代码直观易懂
- **缺点**：涉及字符串操作，性能较慢
- **时间复杂度**：O(n)，n为数字位数
- **空间复杂度**：O(n)

### 2. TransformDigitsOptimized (数学方法)
```go
func TransformDigitsOptimized(num int) int
```
- **原理**：使用数学运算直接处理数字的每一位
- **优点**：性能最优，无额外内存分配
- **缺点**：代码稍复杂
- **时间复杂度**：O(n)，n为数字位数
- **空间复杂度**：O(1)

### 3. TransformDigitsWithDetails (详细信息方法)
```go
func TransformDigitsWithDetails(num int) (int, []int, []int)
```
- **功能**：返回转换结果及原始和转换后的各位数字
- **用途**：调试和详细分析

### 4. 辅助功能
- `BatchTransform`: 批量转换多个数字
- `IsAllSameDigit`: 检查转换后数字是否所有位都相同
- `CountOnesAndZeros`: 统计转换后数字中1和0的个数

## 性能对比

基于基准测试结果（Intel i7-1068NG7 @ 2.30GHz）：

| 算法 | 平均耗时 | 相对性能 |
|------|----------|----------|
| TransformDigitsOptimized | 13.09 ns/op | **最快** |
| TransformDigits | 132.8 ns/op | 10倍慢 |
| TransformDigitsWithDetails | 912.6 ns/op | 70倍慢 |

**推荐**：生产环境使用 `TransformDigitsOptimized`，调试时使用 `TransformDigitsWithDetails`。

## 使用示例

```go
package main

import (
    "fmt"
    "imy/go_docs/algorithm/digit_transform"
)

func main() {
    // 基本使用
    result := digit_transform.TransformDigits(1234)
    fmt.Println(result) // 输出: 1010
    
    // 性能优化版本
    result = digit_transform.TransformDigitsOptimized(1234)
    fmt.Println(result) // 输出: 1010
    
    // 获取详细信息
    result, original, transformed := digit_transform.TransformDigitsWithDetails(1234)
    fmt.Printf("原始: %v, 转换: %v, 结果: %d\n", original, transformed, result)
    // 输出: 原始: [1 2 3 4], 转换: [1 0 1 0], 结果: 1010
    
    // 批量处理
    numbers := []int{1111, 2222, 1234}
    results := digit_transform.BatchTransform(numbers)
    fmt.Println(results) // 输出: [1111 0 1010]
}
```

## 运行测试

```bash
# 运行所有测试
go test -v

# 运行基准测试
go test -bench=.

# 运行示例程序
go run example/main.go
```

## 文件结构

```
digit_transform/
├── digit_transform.go          # 核心算法实现
├── digit_transform_test.go     # 测试用例
├── example/
│   └── main.go                # 使用示例
└── README.md                  # 说明文档
```

## 特殊情况处理

- **零**：`0` → `0`
- **负数**：保持符号，如 `-123` → `-101`
- **单位数**：`7` → `1`, `4` → `0`
- **大数字**：支持 int32 范围内的所有数字

## 算法选择建议

1. **高性能场景**：使用 `TransformDigitsOptimized`
2. **可读性优先**：使用 `TransformDigits`
3. **调试分析**：使用 `TransformDigitsWithDetails`
4. **批量处理**：使用 `BatchTransform`

## 扩展思考

1. **并行处理**：对于大批量数据，可以考虑并行处理
2. **位运算优化**：对于特定场景，可以使用位运算进一步优化
3. **内存池**：对于频繁调用，可以使用对象池减少内存分配
4. **SIMD优化**：对于极高性能要求，可以考虑SIMD指令集优化