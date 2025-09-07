package digit_transform

import (
	"strconv"
	"strings"
)

// TransformDigits 将数字的每一位按奇偶性转换（奇数变1，偶数变0）
// 例如：2222 -> 0, 3333 -> 1111
func TransformDigits(num int) int {
	if num == 0 {
		return 0
	}
	
	// 处理负数
	isNegative := num < 0
	if isNegative {
		num = -num
	}
	
	var result strings.Builder
	numStr := strconv.Itoa(num)
	
	for _, digit := range numStr {
		digitVal := int(digit - '0')
		if digitVal%2 == 1 { // 奇数
			result.WriteString("1")
		} else { // 偶数
			result.WriteString("0")
		}
	}
	
	resultStr := result.String()
	resultNum, _ := strconv.Atoi(resultStr)
	
	if isNegative {
		return -resultNum
	}
	return resultNum
}

// TransformDigitsOptimized 优化版本，使用数学运算而非字符串操作
func TransformDigitsOptimized(num int) int {
	if num == 0 {
		return 0
	}
	
	// 处理负数
	isNegative := num < 0
	if isNegative {
		num = -num
	}
	
	result := 0
	multiplier := 1
	
	for num > 0 {
		digit := num % 10
		if digit%2 == 1 { // 奇数
			result += multiplier
		}
		// 偶数时不加任何值（相当于加0）
		multiplier *= 10
		num /= 10
	}
	
	if isNegative {
		return -result
	}
	return result
}

// TransformDigitsWithDetails 返回转换结果和详细信息
func TransformDigitsWithDetails(num int) (int, []int, []int) {
	if num == 0 {
		return 0, []int{0}, []int{0}
	}
	
	// 处理负数
	isNegative := num < 0
	if isNegative {
		num = -num
	}
	
	var originalDigits []int
	var transformedDigits []int
	
	// 提取数字的每一位
	temp := num
	for temp > 0 {
		digit := temp % 10
		originalDigits = append([]int{digit}, originalDigits...) // 前插保持顺序
		if digit%2 == 1 {
			transformedDigits = append([]int{1}, transformedDigits...)
		} else {
			transformedDigits = append([]int{0}, transformedDigits...)
		}
		temp /= 10
	}
	
	// 计算转换后的数字
	result := 0
	for _, digit := range transformedDigits {
		result = result*10 + digit
	}
	
	if isNegative {
		result = -result
	}
	
	return result, originalDigits, transformedDigits
}

// BatchTransform 批量转换多个数字
func BatchTransform(numbers []int) []int {
	results := make([]int, len(numbers))
	for i, num := range numbers {
		results[i] = TransformDigits(num)
	}
	return results
}

// IsAllSameDigit 检查转换后的数字是否所有位都相同
func IsAllSameDigit(transformedNum int) bool {
	if transformedNum < 0 {
		transformedNum = -transformedNum
	}
	
	if transformedNum < 10 {
		return true
	}
	
	firstDigit := transformedNum % 10
	transformedNum /= 10
	
	for transformedNum > 0 {
		if transformedNum%10 != firstDigit {
			return false
		}
		transformedNum /= 10
	}
	
	return true
}

// CountOnesAndZeros 统计转换后数字中1和0的个数
func CountOnesAndZeros(transformedNum int) (ones, zeros int) {
	if transformedNum < 0 {
		transformedNum = -transformedNum
	}
	
	if transformedNum == 0 {
		return 0, 1
	}
	
	for transformedNum > 0 {
		digit := transformedNum % 10
		if digit == 1 {
			ones++
		} else {
			zeros++
		}
		transformedNum /= 10
	}
	
	return ones, zeros
}