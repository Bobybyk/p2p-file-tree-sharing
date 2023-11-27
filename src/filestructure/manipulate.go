package filestructure

import (
	"fmt"
	"os"
)

func (dir Directory) DumpToDisk(dirName string) {
	err := os.Mkdir(dirName, os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	for _, elem := range dir.Data {

		if chunk, ok := elem.(Chunk); ok { //if the element is a chunk

			f, err := os.OpenFile(dirName+"/"+chunk.Name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println(err)
			}
			defer f.Close()

			_, err = f.Write(chunk.Data)
			if err != nil {
				fmt.Println(err)
				//TODO handle
			}

		} else if dirElem, ok := elem.(Directory); ok { //if the element is a directory

			dirElem.DumpToDisk(dirName + "/" + dirElem.Name)

		} else if big, ok := elem.(Bigfile); ok { //if the element is a bigfile

			big.Dump(dirName + "/" + big.Name)
		}
	}
}

func (big Bigfile) Dump(path string) {
	for i := 0; i < len(big.Data); i++ {

		if newBig, ok := big.Data[i].(Bigfile); ok { //if bigfile -> recursive call
			newBig.Dump(path)
		} else if chunk, ok := big.Data[i].(Chunk); ok { //if chunk -> write
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println(err)
			}
			defer f.Close()

			_, err = f.Write(chunk.Data)
			if err != nil {
				fmt.Println(err)
				//TODO handle
			}
		}
	}
}
