package table_booking

// Group 表示一组客人
type Group struct {
	Size   int // 人数
	Profit int // 收益
}

// Table 表示一张桌子
type Table struct {
	Capacity int // 容量
}

// MaxProfit 计算最大收益 - 经典动态规划解法
// tables: 桌子列表，groups: 客人组列表
func MaxProfit(tables []Table, groups []Group) int {
	if len(tables) == 0 || len(groups) == 0 {
		return 0
	}
	
	// 计算总容量
	totalCapacity := 0
	for _, table := range tables {
		totalCapacity += table.Capacity
	}
	
	// dp[i] 表示容量为i时的最大收益
	dp := make([]int, totalCapacity+1)
	
	// 对每个客人组进行背包处理
	for _, group := range groups {
		// 从后往前遍历，避免重复使用同一组客人
		for capacity := totalCapacity; capacity >= group.Size; capacity-- {
			if dp[capacity-group.Size]+group.Profit > dp[capacity] {
				dp[capacity] = dp[capacity-group.Size] + group.Profit
			}
		}
	}
	
	return dp[totalCapacity]
}

// MaxProfitSimple 最简洁版本 - 01背包问题
func MaxProfitSimple(tableCapacities []int, groupSizes []int, groupProfits []int) int {
	totalCap := 0
	for _, cap := range tableCapacities {
		totalCap += cap
	}
	
	dp := make([]int, totalCap+1)
	// 01背包：每组客人只能选择一次
	for i := 0; i < len(groupSizes); i++ {
		for j := totalCap; j >= groupSizes[i]; j-- {
			if dp[j-groupSizes[i]]+groupProfits[i] > dp[j] {
				dp[j] = dp[j-groupSizes[i]] + groupProfits[i]
			}
		}
	}
	return dp[totalCap]
}

// MaxProfitWithSelection 返回最大收益和选择的客人组
func MaxProfitWithSelection(tables []Table, groups []Group) (int, []int) {
	if len(tables) == 0 || len(groups) == 0 {
		return 0, nil
	}
	
	totalCapacity := 0
	for _, table := range tables {
		totalCapacity += table.Capacity
	}
	
	// dp[i] 表示容量为i时的最大收益
	dp := make([]int, totalCapacity+1)
	// selected[i] 记录容量为i时选择的客人组
	selected := make([][]int, totalCapacity+1)
	for i := range selected {
		selected[i] = make([]int, 0)
	}
	
	for idx, group := range groups {
		for capacity := totalCapacity; capacity >= group.Size; capacity-- {
			newProfit := dp[capacity-group.Size] + group.Profit
			if newProfit > dp[capacity] {
				dp[capacity] = newProfit
				// 复制之前的选择并添加当前组
				selected[capacity] = make([]int, len(selected[capacity-group.Size]))
				copy(selected[capacity], selected[capacity-group.Size])
				selected[capacity] = append(selected[capacity], idx)
			}
		}
	}
	
	return dp[totalCapacity], selected[totalCapacity]
}

// MaxProfitMultipleKnapsack 多个背包版本（每张桌子单独考虑）
func MaxProfitMultipleKnapsack(tables []Table, groups []Group) int {
	if len(tables) == 0 || len(groups) == 0 {
		return 0
	}
	
	// 对每张桌子单独求解01背包问题，然后求和
	totalProfit := 0
	for _, table := range tables {
		// 对当前桌子求解01背包
		dp := make([]int, table.Capacity+1)
		for _, group := range groups {
			for j := table.Capacity; j >= group.Size; j-- {
				if dp[j-group.Size]+group.Profit > dp[j] {
					dp[j] = dp[j-group.Size] + group.Profit
				}
			}
		}
		totalProfit += dp[table.Capacity]
	}
	
	return totalProfit
}

// max 辅助函数
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}