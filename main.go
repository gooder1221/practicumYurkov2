package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Проверка диапазона порта
func validatePort(value interface{}) bool {
	switch v := value.(type) {
	case int:
		return v > 0 && v < 65536
	case int64:
		return v > 0 && v < 65536
	case float64:
		return int(v) > 0 && int(v) < 65536
	default:
		return false
	}
}

// Основная функция валидации YAML
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

	base := filepath.Base(filename)

	// --- Проверка metadata.name ---
	if metadata, ok := raw["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); !ok || name == "" {
			fmt.Printf("%s:4 name is required\n", base)
		}
	}

	// --- Проверка spec.os ---
	if spec, ok := raw["spec"].(map[string]interface{}); ok {
		if osField, ok := spec["os"]; ok {
			if osName, ok := osField.(string); ok {
				if osName != "linux" && osName != "windows" {
					fmt.Printf("%s:10 os has unsupported value '%s'\n", base, osName)
				}
			}
		}

		// --- Проверка containers ---
		if containers, ok := spec["containers"].([]interface{}); ok {
			for _, c := range containers {
				container, ok := c.(map[string]interface{})
				if !ok {
					continue
				}

				// Проверка livenessProbe.httpGet.port
				if probe, ok := container["livenessProbe"].(map[string]interface{}); ok {
					if httpGet, ok := probe["httpGet"].(map[string]interface{}); ok {
						if port, ok := httpGet["port"]; ok {
							if !validatePort(port) {
								fmt.Printf("%s:24 port value out of range\n", base)
							}
						}
					}
				}

				// Проверка resources.requests.cpu
				if resources, ok := container["resources"].(map[string]interface{}); ok {
					if requests, ok := resources["requests"].(map[string]interface{}); ok {
						if cpu, ok := requests["cpu"]; ok {
							switch cpu.(type) {
							case int, int64, float64:
								// всё хорошо
							default:
								fmt.Printf("%s:30 cpu must be int\n", base)
							}
						}
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
