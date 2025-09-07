# 平方和分解问题

## 问题描述

给定一个正整数 n，求最少需要多少个完全平方数的和来表示 n。

例如：
- 12 = 4 + 4 + 4 (3个平方数)
- 13 = 9 + 4 (2个平方数)

## 数学理论基础

### 拉格朗日四平方定理
任何正整数都可以表示为至多4个完全平方数的和。

### 勒让德三平方定理
一个正整数可以表示为3个完全平方数的和，当且仅当它不是形如 4^a(8b+7) 的数。

## 算法实现

### 1. 动态规划解法 (MinSquareSum)

**时间复杂度**: O(n√n)  
**空间复杂度**: O(n)

```go
dp[i] = min(dp[i-j*j] + 1) for all j where j*j <= i
```

### 2. 数学优化解法 (MinSquareSumOptimized)

**时间复杂度**: O(√n)  
**空间复杂度**: O(1)

基于数学定理的优化：
1. 检查是否为完全平方数 → 返回1
2. 检查是否可以用2个平方数表示 → 返回2
3. 检查是否为 4^a(8b+7) 形式 → 返回4
4. 否则返回3

### 3. 带路径返回的解法 (MinSquareSumWithPath)

返回最少平方数个数及具体的分解方案。

### 4. 递归解法

- **BruteForce**: 纯递归，指数时间复杂度
- **BruteForceWithMemo**: 带记忆化的递归，O(n√n)

## 性能对比

基于基准测试结果（n=100）：

| 算法 | 平均耗时 | 相对性能 |
|------|----------|----------|
| MinSquareSumOptimized | 1.955 ns/op | 最快 |
| MinSquareSumWithPath | 1417 ns/op | 中等 |
| MinSquareSum | 1740 ns/op | 中等 |
| BruteForceWithMemo | 4478 ns/op | 较慢 |

## 使用示例

```go
package main

import (
    "fmt"
    "imy/go_docs/algorithm/square_sum"
)

func main() {
    n := 13
    
    // 最优解法
    result := square_sum.MinSquareSumOptimized(n)
    fmt.Printf("%d 最少需要 %d 个平方数\n", n, result)
    
    // 获取具体分解方案
    count, path := square_sum.MinSquareSumWithPath(n)
    fmt.Printf("分解方案: %v (共%d个)\n", path, count)
}
```

## 测试运行

```bash
# 运行所有测试
go test -v

# 运行基准测试
go test -bench=.

# 测试特定函数
go test -run TestMinSquareSum
```

## 文件说明

- `square_sum.go`: 主要算法实现
- `square_sum_test.go`: 完整的测试用例
- `README.md`: 算法说明文档

## 算法选择建议

1. **追求极致性能**: 使用 `MinSquareSumOptimized`
2. **需要分解路径**: 使用 `MinSquareSumWithPath`
3. **学习理解**: 使用 `MinSquareSum` (经典DP)
4. **小规模数据**: 可以使用递归版本进行对比验证

## 扩展思考

1. 如何扩展到求所有可能的最优分解方案？
2. 如何处理更大的数字（如10^9级别）？
3. 能否进一步优化常数因子？

这个问题展示了动态规划、数学定理优化、以及不同算法策略的权衡。