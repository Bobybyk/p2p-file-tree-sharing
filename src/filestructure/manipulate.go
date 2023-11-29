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
	for i := 0; i < len(root.Data); i++ {
		if mt, ok := root.Data[i].(EmptyNode); ok && mt.Hash == hash {
			if mt.Name != "" {
				newChunk.Name = mt.Name
			}
			root.Data[i] = newChunk
		} else if big, ok := root.Data[i].(Bigfile); ok {
			big.addChunk(hash, newChunk)
		} else if dir, ok := root.Data[i].(Directory); ok {
			dir.addChunk(hash, newChunk)
		}
	}
}

func (big *Bigfile) addChunk(hash [32]byte, newChunk Chunk) {
	for i := 0; i < len(big.Data); i++ {
		if mt, ok := big.Data[i].(EmptyNode); ok && mt.Hash == hash {
			if mt.Name != "" {
				newChunk.Name = mt.Name
			}
			big.Data[i] = newChunk
		} else if nextBig, ok := big.Data[i].(Bigfile); ok {
			nextBig.addChunk(hash, newChunk)
		}
	}
}

func (root *Directory) addBigfile(hash [32]byte, newBigfile Bigfile) {

	for i := 0; i < len(root.Data); i++ {

		if mt, ok := root.Data[i].(EmptyNode); ok && mt.Hash == hash { //add to directory
			if mt.Name != "" {
				newBigfile.Name = mt.Name
			}
			root.Data[i] = newBigfile
		} else if big, ok := root.Data[i].(Bigfile); ok { //add to child bigfile
			big.addBigFile(hash, newBigfile)
		} else if dir, ok := root.Data[i].(Directory); ok { // add to child directory
			dir.addBigfile(hash, newBigfile)
		}
	}
}

func (big *Bigfile) addBigFile(hash [32]byte, newbigfile Bigfile) {
	for i := 0; i < len(big.Data); i++ {
		if mt, ok := big.Data[i].(EmptyNode); ok && mt.Hash == hash {
			if mt.Name != "" {
				newbigfile.Name = mt.Name
			}
			big.Data[i] = newbigfile
		} else if nextBig, ok := big.Data[i].(Bigfile); ok {
			nextBig.addBigFile(hash, newbigfile)
		}
	}
}

func (root *Directory) addDirectory(hash [32]byte, newDir Directory) {

	if root.Hash == hash { //change value of root
		root.Data = newDir.Data
	}

	for i := 0; i < len(root.Data); i++ {

		if dir, ok := root.Data[i].(Directory); ok { //add in child directory
			dir.addDirectory(hash, newDir)
		}

		node, ok := root.Data[i].(EmptyNode)
		if ok && node.Hash == hash { //add in child node which type is unknown
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
