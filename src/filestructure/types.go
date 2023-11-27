package filestructure

type File interface{}

type Chunk struct {
	Data []byte
	Name string
}

type Bigfile struct {
	Data []File
	Name string
}

type Directory struct {
	Data []File
	Name string
}

type Root Directory
