package filestructure

type File interface{}

type Child struct {
	Hash [32]byte
	Name string
}

type Node struct {
	Name     string
	Hash     [32]byte
	Data     []File
	Children []Child
}

type Chunk struct {
	Name string
	Hash [32]byte
	Data []byte
}

type Bigfile Node
type Directory Node
type Root Directory
