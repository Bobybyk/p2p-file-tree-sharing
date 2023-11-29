package filestructure

type File interface{}

type Chunk struct {
	Data []byte
	Name string
	Hash [32]byte
}

type Bigfile struct {
	Data []File
	Name string
	Hash [32]byte
}

type Directory struct {
	Name string
	Hash [32]byte
	Data []File
}

type Node struct {
	Name string
	Hash [32]byte
}

type PendingChild struct {
	Name string
	Hash [32]byte
}

type Root Directory
