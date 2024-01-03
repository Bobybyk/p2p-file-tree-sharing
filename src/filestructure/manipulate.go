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
