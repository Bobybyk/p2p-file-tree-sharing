package filestructure

type File interface{}

type EmptyNode struct {
	Name string
	Hash [32]byte
}

type Node struct {
	Name string
	Hash [32]byte
	Data []File
}

type Chunk struct {
	Name string
	Hash [32]byte
	Data []byte
}

type Bigfile Node
type Directory Node
type Root Directory
