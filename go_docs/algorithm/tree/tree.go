package tree

type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

type Tree struct {
	MaxNode *TreeNode
	MinNode *TreeNode
}

func (t *Tree) GetDis(root *TreeNode) int {
	if root == nil {
		return 0
	}
	t.getMaxMin(root)
	if t.MaxNode == nil || t.MinNode == nil {
		return 0
	}
	lcaNode := t.getLCA(root)
	a := t.getNodeDis(lcaNode, t.MaxNode)
	b := t.getNodeDis(lcaNode, t.MinNode)
	return a + b
}

func (t *Tree) getMaxMin(root *TreeNode) {
	if root == nil {
		return
	}
	if t.MinNode == nil || t.MinNode.Val > root.Val {
		t.MinNode = root
	}
	if t.MaxNode == nil || t.MaxNode.Val < root.Val {
		t.MaxNode = root
	}
	t.getMaxMin(root.Left)
	t.getMaxMin(root.Right)
}

func (t *Tree) getLCA(root *TreeNode) *TreeNode {
	if root == nil {
		return root
	}
	if root == t.MaxNode || root == t.MinNode {
		return root
	}

	left := t.getLCA(root.Left)
	right := t.getLCA(root.Right)
	if left != nil && right != nil {
		return root
	}
	if left != nil {
		return left
	}
	if right != nil {
		return right
	}
	return nil
}

func (t *Tree) getNodeDis(root *TreeNode, target *TreeNode) int {
	if root == nil {
		return -1
	}
	if root == target {
		return 0
	}

	d := t.getNodeDis(root.Left, target)
	if d == -1 {
		d = t.getNodeDis(root.Right, target)
	}
	if d != -1 {
		return d + 1
	}
	return -1
}
