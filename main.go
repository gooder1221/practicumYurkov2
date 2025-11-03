package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// validatePort проверяет, что порт в пределах 1–65535
func validatePort(value interface{}) bool {
	port, ok := value.(int)
	if !ok {
		// YAML иногда парсит числа как int64 или float64
		switch v := value.(type) {
		case int64:
			port = int(v)
		case float64:
			port = int(v)
		default:
			return false
		}
	}
	return port > 0 && port < 65536
}

// validateYAML — основная функция валидации YAML файла
func validateYAML(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("%s: unable to read file: %v\n", filename, err)
		return
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		fmt.Printf("YAML decode error: %v\n", err)
		return
	}

	// Проверяем вложенные элементы
	spec, ok := raw["spec"].(map[string]interface{})
	if !ok {
		return
	}

	containers, ok := spec["containers"].([]interface{})
	if !ok {
		return
	}

	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		// Проверяем livenessProbe -> httpGet -> port
		if probe, ok := container["livenessProbe"].(map[string]interface{}); ok {
			if httpGet, ok := probe["httpGet"].(map[string]interface{}); ok {
				if port, ok := httpGet["port"]; ok {
					if !validatePort(port) {
						fmt.Printf("%s:24 port value out of range\n", filepath.Base(filename))
						return
					}
				}
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: yamlvalid <filename>")
		return
	}
	filename := os.Args[1]
	validateYAML(filename)
}
