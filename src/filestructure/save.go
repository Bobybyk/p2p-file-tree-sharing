package filestructure

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func handleBigfile(bigfile Bigfile) ([]byte, error) {
	var data []byte
	for _, child := range bigfile.Data {
		switch child := child.(type) {
		case Chunk:
			data = append(data, child.Data...)
		case Bigfile:
			childData, err := handleBigfile(child)
			if err != nil {
				return nil, err
			}
			data = append(data, childData...)
		default:
			return nil, fmt.Errorf("unexpected type in Bigfile: %T", child)
		}
	}
	return data, nil
}

func SaveFileStructure(path string, node File) error {
	fmt.Println("Saving : ", path)
	switch node := node.(type) {
	case Chunk:
		return os.WriteFile(path, node.Data, 0644)
	case Bigfile:
		data, err := handleBigfile(node)
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0644)
	case Directory:
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
		for _, child := range node.Data {
			switch child := child.(type) {
			case Directory:
				childPath := filepath.Join(path, string(bytes.Trim([]byte(child.Name), "\x00")))
				if err := SaveFileStructure(childPath, child); err != nil {
					return err
				}
			case Chunk:
				childPath := filepath.Join(path, string(bytes.Trim([]byte(child.Name), "\x00")))
				if err := SaveFileStructure(childPath, child); err != nil {
					return err
				}
			case Bigfile:
				childPath := filepath.Join(path, string(bytes.Trim([]byte(child.Name), "\x00")))
				if err := SaveFileStructure(childPath, child); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unexpected type: %T", child)
			}
		}
	default:
		return fmt.Errorf("unexpected type: %T", node)
	}
	return nil
}
