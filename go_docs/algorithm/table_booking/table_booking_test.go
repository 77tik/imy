package table_booking

import (
	"testing"
)

func TestMaxProfit(t *testing.T) {
	tests := []struct {
		name   string
		tables []Table
		groups []Group
		expected int
	}{
		{
			name: "基本案例",
			tables: []Table{{4}, {6}}, // 两张桌子，容量4和6
			groups: []Group{
				{Size: 2, Profit: 100}, // 2人，收益100
				{Size: 3, Profit: 200}, // 3人，收益200
				{Size: 4, Profit: 300}, // 4人，收益300
				{Size: 5, Profit: 400}, // 5人，收益400
			},
			expected: 700, // 选择3人组(200) + 4人组(300) + 2人组(100) = 600，或5人组(400) + 4人组(300) = 700
		},
		{
			name: "单张桌子",
			tables: []Table{{5}},
			groups: []Group{
				{Size: 2, Profit: 100},
				{Size: 3, Profit: 150},
				{Size: 4, Profit: 200},
			},
			expected: 250, // 选择2人组+3人组 = 100+150 = 250
		},
		{
			name: "无法安排",
			tables: []Table{{2}},
			groups: []Group{
				{Size: 3, Profit: 100},
				{Size: 4, Profit: 200},
			},
			expected: 0, // 所有组都太大
		},
		{
			name: "空输入",
			tables: []Table{},
			groups: []Group{},
			expected: 0,
		},
		{
			name: "多个小组合",
			tables: []Table{{3}, {3}, {4}}, // 总容量10
			groups: []Group{
				{Size: 1, Profit: 50},
				{Size: 2, Profit: 120},
				{Size: 3, Profit: 180},
			},
			expected: 350, // 实际最优：1+2+3+3+1 = 50+120+180 = 350
		},
		{
			name: "贪心不是最优",
			tables: []Table{{10}},
			groups: []Group{
				{Size: 6, Profit: 300}, // 性价比50
				{Size: 4, Profit: 250}, // 性价比62.5
				{Size: 5, Profit: 200}, // 性价比40
			},
			expected: 550, // 选择6人组+4人组 = 300+250 = 550
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxProfit(tt.tables, tt.groups)
			if result != tt.expected {
				t.Errorf("MaxProfit() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestMaxProfitSimple(t *testing.T) {
	tests := []struct {
		name           string
		tableCapacities []int
		groupSizes     []int
		groupProfits   []int
		expected       int
	}{
		{
			name: "基本案例",
			tableCapacities: []int{4, 6},
			groupSizes:      []int{2, 3, 4, 5},
			groupProfits:    []int{100, 200, 300, 400},
			expected:        700, // 总容量10，选择2人组+3人组+5人组 = 100+200+400 = 700
		},
		{
			name: "单张桌子",
			tableCapacities: []int{5},
			groupSizes:      []int{2, 3, 4},
			groupProfits:    []int{100, 150, 200},
			expected:        250, // 选择2人组+3人组 = 100+150 = 250
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxProfitSimple(tt.tableCapacities, tt.groupSizes, tt.groupProfits)
			if result != tt.expected {
				t.Errorf("MaxProfitSimple() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestMaxProfitWithSelection(t *testing.T) {
	tables := []Table{{4}, {6}}
	groups := []Group{
		{Size: 2, Profit: 100},
		{Size: 3, Profit: 200},
		{Size: 4, Profit: 300},
		{Size: 5, Profit: 400},
	}

	profit, selection := MaxProfitWithSelection(tables, groups)
	
	if profit != 700 {
		t.Errorf("Expected profit 700, got %d", profit)
	}
	
	// 验证选择的组合是否有效
	totalSize := 0
	totalProfit := 0
	for _, idx := range selection {
		if idx < 0 || idx >= len(groups) {
			t.Errorf("Invalid group index: %d", idx)
			continue
		}
		totalSize += groups[idx].Size
		totalProfit += groups[idx].Profit
	}
	
	totalCapacity := 0
	for _, table := range tables {
		totalCapacity += table.Capacity
	}
	
	if totalSize > totalCapacity {
		t.Errorf("Selected groups exceed capacity: size=%d, capacity=%d", totalSize, totalCapacity)
	}
	
	if totalProfit != profit {
		t.Errorf("Profit mismatch: calculated=%d, returned=%d", totalProfit, profit)
	}
	
	t.Logf("Selected groups: %v, Total profit: %d, Total size: %d", selection, profit, totalSize)
}

func TestMaxProfitMultipleKnapsack(t *testing.T) {
	tests := []struct {
		name   string
		tables []Table
		groups []Group
		expected int
	}{
		{
			name: "基本案例",
			tables: []Table{{4}, {6}},
			groups: []Group{
				{Size: 2, Profit: 100},
				{Size: 3, Profit: 200},
				{Size: 4, Profit: 300},
				{Size: 5, Profit: 400},
			},
			expected: 700, // 选择2人组+3人组+5人组 = 100+200+400 = 700
		},
		{
			name: "单张桌子",
			tables: []Table{{5}},
			groups: []Group{
				{Size: 2, Profit: 100},
				{Size: 3, Profit: 150},
				{Size: 4, Profit: 200},
			},
			expected: 250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxProfitMultipleKnapsack(tt.tables, tt.groups)
			if result != tt.expected {
				t.Errorf("MaxProfitMultipleKnapsack() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

// 测试算法一致性（对于可以用多种方法解决的问题）
func TestAlgorithmConsistency(t *testing.T) {
	tables := []Table{{3}, {4}, {5}}
	groups := []Group{
		{Size: 1, Profit: 50},
		{Size: 2, Profit: 100},
		{Size: 3, Profit: 180},
		{Size: 4, Profit: 250},
	}

	result1 := MaxProfit(tables, groups)
	
	tableCapacities := make([]int, len(tables))
	groupSizes := make([]int, len(groups))
	groupProfits := make([]int, len(groups))
	
	for i, table := range tables {
		tableCapacities[i] = table.Capacity
	}
	for i, group := range groups {
		groupSizes[i] = group.Size
		groupProfits[i] = group.Profit
	}
	
	result2 := MaxProfitSimple(tableCapacities, groupSizes, groupProfits)
	
	if result1 != result2 {
		t.Errorf("Algorithm inconsistency: MaxProfit=%d, MaxProfitSimple=%d", result1, result2)
	}
	
	profit3, _ := MaxProfitWithSelection(tables, groups)
	if result1 != profit3 {
		t.Errorf("Algorithm inconsistency: MaxProfit=%d, MaxProfitWithSelection=%d", result1, profit3)
	}
}

// 边界情况测试
func TestEdgeCases(t *testing.T) {
	t.Run("空桌子列表", func(t *testing.T) {
		result := MaxProfit([]Table{}, []Group{{Size: 2, Profit: 100}})
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})
	
	t.Run("空客人组列表", func(t *testing.T) {
		result := MaxProfit([]Table{{Capacity: 5}}, []Group{})
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})
	
	t.Run("零容量桌子", func(t *testing.T) {
		result := MaxProfit([]Table{{Capacity: 0}}, []Group{{Size: 1, Profit: 100}})
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})
	
	t.Run("零人数组", func(t *testing.T) {
		result := MaxProfit([]Table{{Capacity: 5}}, []Group{{Size: 0, Profit: 100}})
		if result != 100 {
			t.Errorf("Expected 100, got %d", result)
		}
	})
}

// 基准测试
func BenchmarkMaxProfit(b *testing.B) {
	tables := []Table{{4}, {6}, {8}, {10}}
	groups := []Group{
		{Size: 2, Profit: 100},
		{Size: 3, Profit: 200},
		{Size: 4, Profit: 300},
		{Size: 5, Profit: 400},
		{Size: 6, Profit: 500},
		{Size: 7, Profit: 600},
	}
	
	for i := 0; i < b.N; i++ {
		MaxProfit(tables, groups)
	}
}

func BenchmarkMaxProfitSimple(b *testing.B) {
	tableCapacities := []int{4, 6, 8, 10}
	groupSizes := []int{2, 3, 4, 5, 6, 7}
	groupProfits := []int{100, 200, 300, 400, 500, 600}
	
	for i := 0; i < b.N; i++ {
		MaxProfitSimple(tableCapacities, groupSizes, groupProfits)
	}
}

func BenchmarkMaxProfitWithSelection(b *testing.B) {
	tables := []Table{{4}, {6}, {8}}
	groups := []Group{
		{Size: 2, Profit: 100},
		{Size: 3, Profit: 200},
		{Size: 4, Profit: 300},
		{Size: 5, Profit: 400},
	}
	
	for i := 0; i < b.N; i++ {
		MaxProfitWithSelection(tables, groups)
	}
}