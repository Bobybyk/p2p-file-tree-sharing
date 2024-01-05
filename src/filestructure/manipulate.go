package filestructure

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// taille max d'un chunk en octets
const ChunkSize = 1024

// nombre min de fils d'un bigfile
const MinChildren = 2

// nombre max de fils d'un bigfile
const MaxChildren = 32

func ExpandString(name string) string {

	for len(name) < 32 {
		name += "\x00"
	}

	return name
}

// Charge le fichier, à partir du chemin donné, et de ses enfants (si c'est un big file)
func loadFile(path string, name string, data []byte) (File, error) {
	if len(data) <= ChunkSize {
		chunk := Chunk{
			Name: name,
			Data: data,
		}

		hash := sha256.Sum256(append([]byte{0}, chunk.Data...))
		chunk.Hash = hash

		return chunk, nil
	} else {
		bigFile := Bigfile{
			Name: name,
		}

		childSize := (len(data) + MaxChildren - 1) / MaxChildren
		if childSize < ChunkSize {
			childSize = ChunkSize
		}

		for i := 0; i < len(data); i += childSize {
			end := i + childSize
			if end > len(data) {
				end = len(data)
			}

			child, err := loadFile(path, name+fmt.Sprintf(" part %d", i/childSize), data[i:end])
			if err != nil {
				return nil, err
			}

			bigFile.Data = append(bigFile.Data, child)
		}

		var hashTmp []byte

		for _, child := range bigFile.Data {

			if ch, ok := child.(Chunk); ok {
				hashTmp = append(hashTmp, ch.Hash[:]...)
			} else if big, ok := child.(Bigfile); ok {
				hashTmp = append(hashTmp, big.Hash[:]...)
			}
		}

		bigFile.Hash = sha256.Sum256(append([]byte{1}, hashTmp[:]...))

		return bigFile, nil
	}
}

// Charge le répertoire à partir du chemin donné et de ses enfants
func LoadDirectory(path string) (File, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		node := Directory{
			Name: fileInfo.Name(),
		}

		children, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		for _, child := range children {
			childFile, err := LoadDirectory(filepath.Join(path, child.Name()))
			if err != nil {
				return nil, err
			}
			node.Data = append(node.Data, childFile)
		}

		// Compute the hash of the directory
		var hashTmp []byte

		for _, child := range node.Data {

			if ch, ok := child.(Chunk); ok {
				hashTmp = append(hashTmp, []byte(ExpandString(ch.Name))...)
				hashTmp = append(hashTmp, ch.Hash[:]...)
			} else if big, ok := child.(Bigfile); ok {
				hashTmp = append(hashTmp, []byte(ExpandString(big.Name))...)
				hashTmp = append(hashTmp, big.Hash[:]...)
			} else if dir, ok := child.(Directory); ok {
				hashTmp = append(hashTmp, []byte(ExpandString(dir.Name))...)
				hashTmp = append(hashTmp, dir.Hash[:]...)
			}
		}

		node.Hash = sha256.Sum256(append([]byte{2}, hashTmp[:]...))

		return node, nil
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		return loadFile(path, fileInfo.Name(), data)
	}
}

func (root *Node) GetNode(hash [32]byte) File {

	if bytes.Equal(root.Hash[:], hash[:]) {
		return (Directory)(*root)
	}

	for i := 0; i < len(root.Data); i++ {
		if ch, ok := root.Data[i].(Chunk); ok {
			if bytes.Equal(ch.Hash[:], hash[:]) {
				return ch
			}
		} else if dir, ok := root.Data[i].(Directory); ok {
			if bytes.Equal(dir.Hash[:], hash[:]) {
				return dir
			}
		} else if big, ok := root.Data[i].(Bigfile); ok {
			if bytes.Equal(big.Hash[:], hash[:]) {
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
