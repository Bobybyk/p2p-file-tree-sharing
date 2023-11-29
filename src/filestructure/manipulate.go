package filestructure

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

func (root *Directory) addChunk(hash [32]byte, newChunk Chunk) {

}

func (root *Directory) addBigfile(hash [32]byte, newBigfile Bigfile) {

}

func (root *Directory) addDirectory(hash [32]byte, newDir Directory) {

	if root.Hash == hash {
		root.Data = newDir.Data
	}

	for i := 0; i < len(root.Data); i++ {

		if dir, ok := root.Data[i].(Directory); ok {
			dir.addDirectory(hash, newDir)
		}

		node, ok := root.Data[i].(Node)
		if ok && node.Hash == hash {
			newDir.Name = node.Name
			root.Data[i] = newDir
		}
	}
}

func (root *Directory) UpdateDirectory(hash [32]byte, newFile File) {

	if ch, ok := newFile.(Chunk); ok {
		root.addChunk(hash, ch)
	} else if big, ok := newFile.(Bigfile); ok {
		root.addBigfile(hash, big)
	} else if dir, ok := newFile.(Directory); ok {
		root.addDirectory(hash, dir)
	}
}
