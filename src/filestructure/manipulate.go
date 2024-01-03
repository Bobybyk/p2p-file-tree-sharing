package filestructure

func (root *Node) GetNode(hash [32]byte) File {

	if root.Hash == hash {
		return (Directory)(*root)
	}

	for i := 0; i < len(root.Data); i++ {
		if ch, ok := root.Data[i].(Chunk); ok {
			if ch.Hash == hash {
				return ch
			}
		} else if dir, ok := root.Data[i].(Directory); ok {
			if dir.Hash == hash {
				return dir
			}
		} else if big, ok := root.Data[i].(Bigfile); ok {
			if big.Hash == hash {
				return big
			}
		}
	}

	for i := 0; i < len(root.Data); i++ {
		if dir, ok := root.Data[i].(Directory); ok {
			if found := (*Node)(&dir).GetNode(hash); found != nil {
				return found
			}
		} else if big, ok := root.Data[i].(Bigfile); ok {
			if found := (*Node)(&big).GetNode(hash); found != nil {
				return found
			}
		}
	}

	return nil
}
