package index

import "sync"

const degree = 3 

type BTreeKey struct {
	Key    string 
	PageID uint64 
	Offset uint64 
}

type BTreeNode struct {
	Keys     []BTreeKey
	Children []*BTreeNode
	IsLeaf   bool
}

func newNode(isLeaf bool) *BTreeNode {
	return &BTreeNode{IsLeaf: isLeaf}
}

type BTree struct {
	mu   sync.RWMutex
	root *BTreeNode
}

func NewBTree() *BTree {
	return &BTree{root: newNode(true)}
}

func (t *BTree) Search(key string) *BTreeKey {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return search(t.root, key)
}

func search(node *BTreeNode, key string) *BTreeKey {
	i := 0
	for i < len(node.Keys) && key > node.Keys[i].Key {
		i++
	}
	if i < len(node.Keys) && key == node.Keys[i].Key {
		return &node.Keys[i]
	}
	if node.IsLeaf {
		return nil
	}
	return search(node.Children[i], key)
}

func (t *BTree) Insert(key BTreeKey) {
	t.mu.Lock()
	defer t.mu.Unlock()

	root := t.root
	if len(root.Keys) == 2*degree-1 {
		newRoot := newNode(false)
		newRoot.Children = append(newRoot.Children, t.root)
		splitChild(newRoot, 0)
		t.root = newRoot
	}
	insertNonFull(t.root, key)
}

func insertNonFull(node *BTreeNode, key BTreeKey) {
	i := len(node.Keys) - 1
	if node.IsLeaf {
		node.Keys = append(node.Keys, BTreeKey{})
		for i >= 0 && key.Key < node.Keys[i].Key {
			node.Keys[i+1] = node.Keys[i]
			i--
		}
		// Update if key already exists
		if i >= 0 && key.Key == node.Keys[i].Key {
			node.Keys[i] = key
			node.Keys = node.Keys[:len(node.Keys)-1]
			return
		}
		node.Keys[i+1] = key
		return
	}
	for i >= 0 && key.Key < node.Keys[i].Key {
		i--
	}
	i++
	if len(node.Children[i].Keys) == 2*degree-1 {
		splitChild(node, i)
		if key.Key > node.Keys[i].Key {
			i++
		}
	}
	insertNonFull(node.Children[i], key)
}

func splitChild(parent *BTreeNode, i int) {
	t := degree
	child := parent.Children[i]
	sibling := newNode(child.IsLeaf)

	medianKey := child.Keys[t-1]

	sibling.Keys = append(sibling.Keys, child.Keys[t:]...)
	child.Keys = child.Keys[:t-1]

	if !child.IsLeaf {
		sibling.Children = append(sibling.Children, child.Children[t:]...)
		child.Children = child.Children[:t]
	}

	parent.Keys = append(parent.Keys, BTreeKey{})
	copy(parent.Keys[i+1:], parent.Keys[i:])
	parent.Keys[i] = medianKey

	parent.Children = append(parent.Children, nil)
	copy(parent.Children[i+2:], parent.Children[i+1:])
	parent.Children[i+1] = sibling
}

func (t *BTree) Delete(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	deleteKey(t.root, key)
}

func deleteKey(node *BTreeNode, key string) {
	i := 0
	for i < len(node.Keys) && key > node.Keys[i].Key {
		i++
	}
	if i < len(node.Keys) && key == node.Keys[i].Key {
		if node.IsLeaf {
			node.Keys = append(node.Keys[:i], node.Keys[i+1:]...)
		}
		return
	}
	if !node.IsLeaf {
		deleteKey(node.Children[i], key)
	}
}

func (t *BTree) RangeSearch(startKey, endKey string) []BTreeKey {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var results []BTreeKey
	rangeSearch(t.root, startKey, endKey, &results)
	return results
}

func rangeSearch(node *BTreeNode, start, end string, results *[]BTreeKey) {
	for i := 0; i < len(node.Keys); i++ {
		if !node.IsLeaf {
			rangeSearch(node.Children[i], start, end, results)
		}
		if node.Keys[i].Key >= start && node.Keys[i].Key <= end {
			*results = append(*results, node.Keys[i])
		}
	}
	if !node.IsLeaf {
		rangeSearch(node.Children[len(node.Keys)], start, end, results)
	}
}
