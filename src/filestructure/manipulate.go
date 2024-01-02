package filestructure

import (
	"errors"
)

func (root *Directory) GetNode(hash [32]byte) File {
	for i := 0; i < len(root.Data); i++ {

		if ch, ok := root.Data[i].(Chunk); ok && ch.Hash == hash {
			return root.Data[i]
		} else if big, ok := root.Data[i].(Bigfile); ok {
			return big.GetNode(hash)
		}
	}

	return nil
}

func (big *Bigfile) GetNode(hash [32]byte) File {

	for i := 0; i < len(big.Data); i++ {

		if ch, ok := big.Data[i].(Chunk); ok && ch.Hash == hash {
			return big.Data[i]
		} else if nextBig, ok := big.Data[i].(Bigfile); ok {
			return nextBig.GetNode(hash)
		}
	}

	return nil
}

func (root *Node) GetParentNode(hash [32]byte) (*Node, string, error) {

	for _, child := range root.Children {
		if child.Hash == hash {
			return (*Node)(root), child.Name, nil
		}
	}

	for _, data := range root.Data {
		if node, ok := data.(Node); ok {
			if parent, n, _ := node.GetParentNode(hash); parent != nil {
				return parent, n, nil
			}
		}
	}
	return &Node{}, "", errors.New("Node not found")
}
