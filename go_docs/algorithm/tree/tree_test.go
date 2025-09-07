package tree

import (
	"testing"
)

// 创建测试用的二叉树节点
func createNode(val int) *TreeNode {
	return &TreeNode{Val: val}
}

// 测试用例1：简单的三节点树
func TestGetDis_SimpleTree(t *testing.T) {
	// 构建树：
	//     2
	//    / \
	//   1   3
	root := createNode(2)
	root.Left = createNode(1)
	root.Right = createNode(3)

	tree := &Tree{}
	result := tree.GetDis(root)
	expected := 2 // 从节点1到节点3的距离

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试用例2：更复杂的树结构
func TestGetDis_ComplexTree(t *testing.T) {
	// 构建树：
	//       4
	//      / \
	//     2   6
	//    / \ / \
	//   1  3 5  7
	root := createNode(4)
	root.Left = createNode(2)
	root.Right = createNode(6)
	root.Left.Left = createNode(1)
	root.Left.Right = createNode(3)
	root.Right.Left = createNode(5)
	root.Right.Right = createNode(7)

	tree := &Tree{}
	result := tree.GetDis(root)
	expected := 4 // 从节点1到节点7的距离：1->2->4->6->7

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试用例3：单节点树
func TestGetDis_SingleNode(t *testing.T) {
	root := createNode(5)

	tree := &Tree{}
	result := tree.GetDis(root)
	expected := 0 // 只有一个节点，最大最小都是同一个节点

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试用例4：空树
func TestGetDis_EmptyTree(t *testing.T) {
	tree := &Tree{}
	result := tree.GetDis(nil)
	expected := 0

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试用例5：左偏树
func TestGetDis_LeftSkewedTree(t *testing.T) {
	// 构建树：
	//   5
	//  /
	// 3
	///
	//1
	root := createNode(5)
	root.Left = createNode(3)
	root.Left.Left = createNode(1)

	tree := &Tree{}
	result := tree.GetDis(root)
	expected := 2 // 从节点1到节点5的距离

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试用例6：右偏树
func TestGetDis_RightSkewedTree(t *testing.T) {
	// 构建树：
	// 1
	//  \
	//   3
	//    \
	//     5
	root := createNode(1)
	root.Right = createNode(3)
	root.Right.Right = createNode(5)

	tree := &Tree{}
	result := tree.GetDis(root)
	expected := 2 // 从节点1到节点5的距离

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试用例7：包含重复值的树
func TestGetDis_DuplicateValues(t *testing.T) {
	// 构建树：
	//     2
	//    / \
	//   2   3
	//  /
	// 1
	root := createNode(2)
	root.Left = createNode(2)
	root.Right = createNode(3)
	root.Left.Left = createNode(1)

	tree := &Tree{}
	result := tree.GetDis(root)
	expected := 3 // 从节点1到节点3的距离

	if result != expected {
		t.Errorf("Expected %d, but got %d", expected, result)
	}
}

// 测试辅助函数：getMaxMin
func TestGetMaxMin(t *testing.T) {
	root := createNode(4)
	root.Left = createNode(2)
	root.Right = createNode(6)
	root.Left.Left = createNode(1)
	root.Left.Right = createNode(3)
	root.Right.Left = createNode(5)
	root.Right.Right = createNode(7)

	tree := &Tree{}
	tree.getMaxMin(root)

	if tree.MinNode.Val != 1 {
		t.Errorf("Expected MinNode value to be 1, but got %d", tree.MinNode.Val)
	}
	if tree.MaxNode.Val != 7 {
		t.Errorf("Expected MaxNode value to be 7, but got %d", tree.MaxNode.Val)
	}
}

// 测试辅助函数：getLCA
func TestGetLCA(t *testing.T) {
	root := createNode(4)
	root.Left = createNode(2)
	root.Right = createNode(6)
	root.Left.Left = createNode(1)
	root.Left.Right = createNode(3)
	root.Right.Left = createNode(5)
	root.Right.Right = createNode(7)

	tree := &Tree{}
	tree.MaxNode = root.Right.Right // 节点7
	tree.MinNode = root.Left.Left   // 节点1

	lca := tree.getLCA(root)
	if lca.Val != 4 {
		t.Errorf("Expected LCA value to be 4, but got %d", lca.Val)
	}
}

// 测试辅助函数：getNodeDis
func TestGetNodeDis(t *testing.T) {
	root := createNode(4)
	root.Left = createNode(2)
	root.Right = createNode(6)
	root.Left.Left = createNode(1)

	tree := &Tree{}
	distance := tree.getNodeDis(root, root.Left.Left)
	expected := 2 // 从根节点4到节点1的距离

	if distance != expected {
		t.Errorf("Expected distance to be %d, but got %d", expected, distance)
	}

	// 测试目标节点不存在的情况
	nonExistentNode := createNode(99)
	distance = tree.getNodeDis(root, nonExistentNode)
	if distance != -1 {
		t.Errorf("Expected distance to be -1 for non-existent node, but got %d", distance)
	}
}

// 基准测试
func BenchmarkGetDis(b *testing.B) {
	// 构建一个较大的平衡二叉树
	root := createNode(8)
	root.Left = createNode(4)
	root.Right = createNode(12)
	root.Left.Left = createNode(2)
	root.Left.Right = createNode(6)
	root.Right.Left = createNode(10)
	root.Right.Right = createNode(14)
	root.Left.Left.Left = createNode(1)
	root.Left.Left.Right = createNode(3)
	root.Left.Right.Left = createNode(5)
	root.Left.Right.Right = createNode(7)
	root.Right.Left.Left = createNode(9)
	root.Right.Left.Right = createNode(11)
	root.Right.Right.Left = createNode(13)
	root.Right.Right.Right = createNode(15)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree := &Tree{}
		tree.GetDis(root)
	}
}