package main

import (
	"fmt"
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

// Основная функция проверки YAML
func validateYAML(filename string) {
	data, err := os.ReadFile(filename) // ✅ заменили ioutil.ReadFile
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

	// --- metadata.name ---
	if metadata, ok := raw["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); !ok || name == "" {
			fmt.Printf("%s:4 name is required\n", base)
		}
	}

	// --- spec ---
	if spec, ok := raw["spec"].(map[string]interface{}); ok {
		// --- spec.os ---
		if osField, ok := spec["os"]; ok {
			if osName, ok := osField.(string); ok {
				if osName != "linux" && osName != "windows" {
					fmt.Printf("%s:10 os has unsupported value '%s'\n", base, osName)
				}
			}
		}

		// --- spec.containers ---
		if containers, ok := spec["containers"].([]interface{}); ok {
			for _, c := range containers {
				container, ok := c.(map[string]interface{})
				if !ok {
					continue
				}

				// --- container.name ---
				if name, ok := container["name"].(string); !ok || name == "" {
					fmt.Printf("%s:12 name is required\n", base)
				}

				// --- container.ports[].containerPort ---
				if ports, ok := container["ports"].([]interface{}); ok {
					for _, p := range ports {
						if portObj, ok := p.(map[string]interface{}); ok {
							if port, ok := portObj["containerPort"]; ok {
								if !validatePort(port) {
									fmt.Printf("%s:15 containerPort value out of range\n", base)
								}
							}
						}
					}
				}

				// --- readinessProbe.httpGet.port ---
				if probe, ok := container["readinessProbe"].(map[string]interface{}); ok {
					if httpGet, ok := probe["httpGet"].(map[string]interface{}); ok {
						if port, ok := httpGet["port"]; ok {
							if !validatePort(port) {
								fmt.Printf("%s:20 port value out of range\n", base)
							}
						}
					}
				}

				// --- livenessProbe.httpGet.port ---
				if probe, ok := container["livenessProbe"].(map[string]interface{}); ok {
					if httpGet, ok := probe["httpGet"].(map[string]interface{}); ok {
						if port, ok := httpGet["port"]; ok {
							if !validatePort(port) {
								fmt.Printf("%s:24 port value out of range\n", base)
							}
						}
					}
				}

				// --- resources ---
				if resources, ok := container["resources"].(map[string]interface{}); ok {
					// --- limits.cpu ---
					if limits, ok := resources["limits"].(map[string]interface{}); ok {
						if cpu, ok := limits["cpu"]; ok {
							switch cpu.(type) {
							case int, int64, float64:
								// OK
							default:
								fmt.Printf("%s:27 cpu must be int\n", base)
							}
						}
					}

					// --- requests.cpu ---
					if requests, ok := resources["requests"].(map[string]interface{}); ok {
						if cpu, ok := requests["cpu"]; ok {
							switch cpu.(type) {
							case int, int64, float64:
								// OK
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
