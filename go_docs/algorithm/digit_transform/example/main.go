package main

import (
	"fmt"
	"imy/go_docs/algorithm/digit_transform"
)

func main() {
	fmt.Println("=== 数字位奇偶转换算法演示 ===")
	fmt.Println("规则：奇数位变为1，偶数位变为0")
	fmt.Println()

	// 基本示例
	testCases := []int{2222, 3333, 1234, 5678, 0, 7, 4, -123, 987654321}

	fmt.Println("1. 基本转换示例：")
	for _, num := range testCases {
		result := digit_transform.TransformDigits(num)
		fmt.Printf("输入: %10d -> 输出: %10d\n", num, result)
	}

	fmt.Println("\n2. 详细转换过程：")
	detailCases := []int{1234, 5678, 9876}
	for _, num := range detailCases {
		result, original, transformed := digit_transform.TransformDigitsWithDetails(num)
		fmt.Printf("输入: %d\n", num)
		fmt.Printf("  原始各位: %v\n", original)
		fmt.Printf("  转换各位: %v\n", transformed)
		fmt.Printf("  最终结果: %d\n", result)
		fmt.Println()
	}

	fmt.Println("3. 批量转换：")
	batchInput := []int{1111, 2222, 1234, 5678}
	batchResult := digit_transform.BatchTransform(batchInput)
	fmt.Printf("输入: %v\n", batchInput)
	fmt.Printf("输出: %v\n", batchResult)
	fmt.Println()

	fmt.Println("4. 转换结果分析：")
	analysisCases := []int{1010, 1111, 0, 101}
	for _, num := range analysisCases {
		isAllSame := digit_transform.IsAllSameDigit(num)
		ones, zeros := digit_transform.CountOnesAndZeros(num)
		fmt.Printf("数字 %d: 全相同=%t, 1的个数=%d, 0的个数=%d\n", 
			num, isAllSame, ones, zeros)
	}

	fmt.Println("\n5. 性能对比演示：")
	testNum := 123456789
	fmt.Printf("测试数字: %d\n", testNum)
	
	// 字符串方法
	result1 := digit_transform.TransformDigits(testNum)
	fmt.Printf("字符串方法结果: %d\n", result1)
	
	// 优化数学方法
	result2 := digit_transform.TransformDigitsOptimized(testNum)
	fmt.Printf("数学方法结果: %d\n", result2)
	
	fmt.Printf("结果一致性: %t\n", result1 == result2)

	fmt.Println("\n=== 演示完成 ===")
	fmt.Println("提示：运行 'go test -bench=.' 查看性能基准测试")
}