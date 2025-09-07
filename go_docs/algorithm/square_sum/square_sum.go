package square_sum

import (
	"math"
)

// MinSquareSum 使用动态规划计算将数字n分解为最少个完全平方数的和
// 时间复杂度: O(n * sqrt(n))
// 空间复杂度: O(n)
func MinSquareSum(n int) int {
	if n <= 0 {
		return 0
	}
	
	// dp[i] 表示数字i最少可以分解为多少个完全平方数的和
	dp := make([]int, n+1)
	
	// 初始化：每个数字最多需要i个1的平方
	for i := 1; i <= n; i++ {
		dp[i] = i
	}
	
	// 动态规划：对于每个数字i，尝试所有可能的完全平方数j*j
	for i := 1; i <= n; i++ {
		for j := 1; j*j <= i; j++ {
			dp[i] = min(dp[i], dp[i-j*j]+1)
		}
	}
	
	return dp[n]
}

// MinSquareSumOptimized 优化版本，使用数学定理进行预处理
// 拉格朗日四平方定理：任何正整数都可以表示为至多4个完全平方数的和
// 勒让德三平方定理：当且仅当n不是4^a(8b+7)的形式时，n可以表示为3个完全平方数的和
func MinSquareSumOptimized(n int) int {
	if n <= 0 {
		return 0
	}
	
	// 检查是否为完全平方数
	if isSquare(n) {
		return 1
	}
	
	// 检查是否可以表示为两个完全平方数的和
	for i := 1; i*i <= n; i++ {
		if isSquare(n - i*i) {
			return 2
		}
	}
	
	// 检查是否为4^a(8b+7)的形式，如果是则需要4个平方数
	if isFormOf4a8b7(n) {
		return 4
	}
	
	// 否则需要3个平方数
	return 3
}

// MinSquareSumWithPath 返回最少个数和具体的分解路径
func MinSquareSumWithPath(n int) (int, []int) {
	if n <= 0 {
		return 0, nil
	}
	
	dp := make([]int, n+1)
	parent := make([]int, n+1) // 记录路径
	
	for i := 1; i <= n; i++ {
		dp[i] = i
		parent[i] = 1 // 默认使用1的平方
	}
	
	for i := 1; i <= n; i++ {
		for j := 1; j*j <= i; j++ {
			if dp[i-j*j]+1 < dp[i] {
				dp[i] = dp[i-j*j] + 1
				parent[i] = j * j
			}
		}
	}
	
	// 重构路径
	path := []int{}
	current := n
	for current > 0 {
		square := parent[current]
		path = append(path, square)
		current -= square
	}
	
	return dp[n], path
}

// 辅助函数：检查是否为完全平方数
func isSquare(n int) bool {
	if n < 0 {
		return false
	}
	sqrt := int(math.Sqrt(float64(n)))
	return sqrt*sqrt == n
}

// 辅助函数：检查是否为4^a(8b+7)的形式
func isFormOf4a8b7(n int) bool {
	// 不断除以4，直到不能整除
	for n%4 == 0 {
		n /= 4
	}
	// 检查是否为8k+7的形式
	return n%8 == 7
}

// 辅助函数：返回两个数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BruteForce 暴力递归解法（仅用于小数字验证）
func BruteForce(n int) int {
	if n <= 0 {
		return 0
	}
	if isSquare(n) {
		return 1
	}
	
	minCount := n // 最坏情况：n个1的平方
	for i := 1; i*i < n; i++ {
		count := 1 + BruteForce(n-i*i)
		minCount = min(minCount, count)
	}
	
	return minCount
}

// BruteForceWithMemo 带记忆化的递归解法
func BruteForceWithMemo(n int) int {
	memo := make(map[int]int)
	return bruteForceHelper(n, memo)
}

func bruteForceHelper(n int, memo map[int]int) int {
	if n <= 0 {
		return 0
	}
	if val, exists := memo[n]; exists {
		return val
	}
	if isSquare(n) {
		memo[n] = 1
		return 1
	}
	
	minCount := n
	for i := 1; i*i < n; i++ {
		count := 1 + bruteForceHelper(n-i*i, memo)
		minCount = min(minCount, count)
	}
	
	memo[n] = minCount
	return minCount
}