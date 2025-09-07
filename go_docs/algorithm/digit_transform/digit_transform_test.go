package digit_transform

import (
	"reflect"
	"testing"
)

func TestTransformDigits(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "示例1: 2222",
			input:    2222,
			expected: 0, // 2是偶数，全部变为0，0000 = 0
		},
		{
			name:     "示例2: 3333",
			input:    3333,
			expected: 1111, // 3是奇数，全部变为1
		},
		{
			name:     "零",
			input:    0,
			expected: 0,
		},
		{
			name:     "单位数奇数",
			input:    7,
			expected: 1,
		},
		{
			name:     "单位数偶数",
			input:    4,
			expected: 0,
		},
		{
			name:     "混合数字",
			input:    1234,
			expected: 1010, // 1(奇)->1, 2(偶)->0, 3(奇)->1, 4(偶)->0
		},
		{
			name:     "全奇数",
			input:    1357,
			expected: 1111,
		},
		{
			name:     "全偶数",
			input:    2468,
			expected: 0, // 0000 = 0
		},
		{
			name:     "负数",
			input:    -1234,
			expected: -1010,
		},
		{
			name:     "大数字",
			input:    987654321,
			expected: 101010101, // 9,8,7,6,5,4,3,2,1 -> 1,0,1,0,1,0,1,0,1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformDigits(tt.input)
			if result != tt.expected {
				t.Errorf("TransformDigits(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransformDigitsOptimized(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"示例1: 2222", 2222, 0},
		{"示例2: 3333", 3333, 1111},
		{"零", 0, 0},
		{"混合数字", 1234, 1010},
		{"负数", -1234, -1010},
		{"大数字", 987654321, 101010101},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransformDigitsOptimized(tt.input)
			if result != tt.expected {
				t.Errorf("TransformDigitsOptimized(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransformDigitsWithDetails(t *testing.T) {
	tests := []struct {
		name                string
		input               int
		expectedResult      int
		expectedOriginal    []int
		expectedTransformed []int
	}{
		{
			name:                "示例: 1234",
			input:               1234,
			expectedResult:      1010,
			expectedOriginal:    []int{1, 2, 3, 4},
			expectedTransformed: []int{1, 0, 1, 0},
		},
		{
			name:                "零",
			input:               0,
			expectedResult:      0,
			expectedOriginal:    []int{0},
			expectedTransformed: []int{0},
		},
		{
			name:                "单位数",
			input:               7,
			expectedResult:      1,
			expectedOriginal:    []int{7},
			expectedTransformed: []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, original, transformed := TransformDigitsWithDetails(tt.input)
			if result != tt.expectedResult {
				t.Errorf("TransformDigitsWithDetails(%d) result = %d, expected %d", tt.input, result, tt.expectedResult)
			}
			if !reflect.DeepEqual(original, tt.expectedOriginal) {
				t.Errorf("TransformDigitsWithDetails(%d) original = %v, expected %v", tt.input, original, tt.expectedOriginal)
			}
			if !reflect.DeepEqual(transformed, tt.expectedTransformed) {
				t.Errorf("TransformDigitsWithDetails(%d) transformed = %v, expected %v", tt.input, transformed, tt.expectedTransformed)
			}
		})
	}
}

func TestBatchTransform(t *testing.T) {
	input := []int{1234, 5678, 0, -123}
	expected := []int{1010, 1010, 0, -101}
	
	result := BatchTransform(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("BatchTransform(%v) = %v, expected %v", input, result, expected)
	}
}

func TestIsAllSameDigit(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected bool
	}{
		{"全1", 1111, true},
		{"全0（单个0）", 0, true},
		{"混合", 1010, false},
		{"单位数", 5, true},
		{"负数全1", -1111, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAllSameDigit(tt.input)
			if result != tt.expected {
				t.Errorf("IsAllSameDigit(%d) = %t, expected %t", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCountOnesAndZeros(t *testing.T) {
	tests := []struct {
		name          string
		input         int
		expectedOnes  int
		expectedZeros int
	}{
		{"1010", 1010, 2, 2},
		{"1111", 1111, 4, 0},
		{"0", 0, 0, 1},
		{"101", 101, 2, 1},
		{"负数", -1010, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ones, zeros := CountOnesAndZeros(tt.input)
			if ones != tt.expectedOnes || zeros != tt.expectedZeros {
				t.Errorf("CountOnesAndZeros(%d) = (%d, %d), expected (%d, %d)", 
					tt.input, ones, zeros, tt.expectedOnes, tt.expectedZeros)
			}
		})
	}
}

// 算法一致性测试
func TestAlgorithmConsistency(t *testing.T) {
	testCases := []int{0, 1, 12, 123, 1234, 5678, 9876543210, -123, -456}
	
	for _, num := range testCases {
		result1 := TransformDigits(num)
		result2 := TransformDigitsOptimized(num)
		
		if result1 != result2 {
			t.Errorf("Algorithm inconsistency for %d: TransformDigits=%d, TransformDigitsOptimized=%d", 
				num, result1, result2)
		}
	}
}

// 边界情况测试
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input int
	}{
		{"最大int32", 2147483647},
		{"最小int32", -2147483648},
		{"全9", 999999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 只要不panic就算通过
			result1 := TransformDigits(tt.input)
			result2 := TransformDigitsOptimized(tt.input)
			
			if result1 != result2 {
				t.Errorf("Edge case inconsistency for %d: %d vs %d", tt.input, result1, result2)
			}
		})
	}
}

// 基准测试
func BenchmarkTransformDigits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TransformDigits(123456789)
	}
}

func BenchmarkTransformDigitsOptimized(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TransformDigitsOptimized(123456789)
	}
}

func BenchmarkTransformDigitsWithDetails(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TransformDigitsWithDetails(123456789)
	}
}