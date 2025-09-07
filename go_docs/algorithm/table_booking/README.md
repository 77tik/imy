# 桌子预订收益最大化问题

## 问题描述

有n个桌子，每个桌子可以容纳若干人。有m组客人，每组有若干个人，每组可以带来若干收益。每组客人不跟其他组拼桌。求最大收益。

这是一个经典的**01背包问题**，其中：
- 桌子的总容量是背包容量
- 每组客人是物品，有人数（重量）和收益（价值）
- 每组客人只能选择一次（01背包特性）

## 算法实现

### 1. MaxProfit - 经典动态规划解法

```go
func MaxProfit(tables []Table, groups []Group) int
```

**算法原理：**
- 使用01背包动态规划算法
- `dp[i]` 表示容量为i时的最大收益
- 状态转移方程：`dp[j] = max(dp[j], dp[j-size] + profit)`
- 从后往前遍历避免重复使用同一组客人

**时间复杂度：** O(总容量 × 客人组数)  
**空间复杂度：** O(总容量)

### 2. MaxProfitSimple - 最简洁版本

```go
func MaxProfitSimple(tableCapacities []int, groupSizes []int, groupProfits []int) int
```

**特点：**
- 参数更简洁，直接使用数组
- 算法逻辑与MaxProfit相同
- 适合快速实现和理解

### 3. MaxProfitWithSelection - 带路径返回

```go
func MaxProfitWithSelection(tables []Table, groups []Group) (int, []int)
```

**特点：**
- 不仅返回最大收益，还返回选择的客人组索引
- 使用额外的二维数组记录选择路径
- 适合需要知道具体选择方案的场景

**时间复杂度：** O(总容量 × 客人组数)  
**空间复杂度：** O(总容量 × 客人组数)

### 4. MaxProfitMultipleKnapsack - 多背包版本

```go
func MaxProfitMultipleKnapsack(tables []Table, groups []Group) int
```

**算法原理：**
- 将问题分解为多个独立的01背包问题
- 对每张桌子单独求解最优方案
- 将各桌子的最优收益相加

**适用场景：** 当桌子之间完全独立，不能跨桌安排时

## 性能对比

基于基准测试结果：

| 算法 | 平均执行时间 | 相对性能 |
|------|-------------|----------|
| MaxProfit | 251.6 ns/op | 基准 |
| MaxProfitSimple | 255.2 ns/op | 相当 |
| MaxProfitWithSelection | 3145 ns/op | 较慢（需要记录路径） |

## 使用示例

```go
package main

import (
    "fmt"
    "imy/go_docs/algorithm/table_booking"
)

func main() {
    // 定义桌子
    tables := []table_booking.Table{
        {Capacity: 4}, // 4人桌
        {Capacity: 6}, // 6人桌
    }
    
    // 定义客人组
    groups := []table_booking.Group{
        {Size: 2, Profit: 100}, // 2人组，收益100
        {Size: 3, Profit: 200}, // 3人组，收益200
        {Size: 4, Profit: 300}, // 4人组，收益300
        {Size: 5, Profit: 400}, // 5人组，收益400
    }
    
    // 求解最大收益
    maxProfit := table_booking.MaxProfit(tables, groups)
    fmt.Printf("最大收益: %d\n", maxProfit) // 输出: 700
    
    // 获取具体选择方案
    profit, selected := table_booking.MaxProfitWithSelection(tables, groups)
    fmt.Printf("收益: %d, 选择的组: %v\n", profit, selected)
}
```

## 测试运行

```bash
# 运行所有测试
go test -v

# 运行基准测试
go test -bench=.
```

## 文件说明

- `table_booking.go` - 主要算法实现
- `table_booking_test.go` - 测试用例和基准测试
- `README.md` - 说明文档

## 算法选择建议

1. **一般情况：** 使用 `MaxProfit` 或 `MaxProfitSimple`
2. **需要知道具体方案：** 使用 `MaxProfitWithSelection`
3. **桌子完全独立：** 使用 `MaxProfitMultipleKnapsack`
4. **追求极致性能：** 使用 `MaxProfit`

## 扩展思考

1. **完全背包变种：** 如果允许同一组客人多次预订，可改为完全背包
2. **多维背包：** 如果考虑时间段等多个约束条件
3. **分组背包：** 如果客人组之间有互斥关系
4. **在线算法：** 如果客人组动态到达，需要在线决策