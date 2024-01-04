package filestructure

import (
	"fmt"
	"strconv"
)

// Print the file structure
func PrintFileStructure(file File, indent string, simplified bool) {
	switch f := file.(type) {
	case Directory:
		fmt.Println(indent + f.Name + "/")
		for _, child := range f.Data {
			PrintFileStructure(child, indent+"  ", simplified)
		}
	case Chunk:
		fmt.Println(indent + f.Name)
	case Bigfile:
		fmt.Println(indent + f.Name + " (bigfile)")
		if simplified {
			fmt.Println(indent + "  nombre de fils: " + strconv.Itoa(len(f.Data)))
		} else {
			for _, child := range f.Data {
				PrintFileStructure(child, indent+"  ", simplified)
			}
		}
	default:
		fmt.Println("Unknown file type")
	}
}
