package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------- Основные структуры ----------

// Pod — верхний уровень
type Pod struct {
	APIVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Metadata   ObjectMeta `yaml:"metadata"`
	Spec       PodSpec   `yaml:"spec"`
}

// ObjectMeta — метаданные пода
type ObjectMeta struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

// PodSpec — описание пода
type PodSpec struct {
	OS         *PodOS      `yaml:"os"`
	Containers []Container `yaml:"containers"`
}

// PodOS — операционная система пода
type PodOS struct {
	Name string `yaml:"name"`
}

// Container — описание контейнера
type Container struct {
	Name           string               `yaml:"name"`
	Image          string               `yaml:"image"`
	Ports          *ContainerPort       `yaml:"ports"`
	ReadinessProbe *Probe               `yaml:"readinessProbe"`
	LivenessProbe  *Probe               `yaml:"livenessProbe"`
	Resources      ResourceRequirements `yaml:"resources"`
}

// ContainerPort — описание порта
type ContainerPort struct {
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol"`
}

// Probe — проверка готовности/живости
type Probe struct {
	HTTPGet HTTPGetAction `yaml:"httpGet"`
}

// HTTPGetAction — HTTP GET действие
type HTTPGetAction struct {
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
}

// ResourceRequirements — требования к ресурсам
type ResourceRequirements struct {
	Requests map[string]string `yaml:"requests"`
	Limits   map[string]string `yaml:"limits"`
}

// ---------- Валидация ----------

func (p *Pod) Validate() []error {
	var errs []error

	// 1. Верхний уровень
	if p.APIVersion != "v1" {
		errs = append(errs, errors.New("apiVersion must be 'v1'"))
	}
	if p.Kind != "Pod" {
		errs = append(errs, errors.New("kind must be 'Pod'"))
	}
	if p.Metadata.Name == "" {
		errs = append(errs, errors.New("metadata.name is required"))
	}
	// 2. PodSpec
	if len(p.Spec.Containers) == 0 {
		errs = append(errs, errors.New("spec.containers must not be empty"))
	}
	if p.Spec.OS != nil {
		if p.Spec.OS.Name != "linux" && p.Spec.OS.Name != "windows" {
			errs = append(errs, errors.New("spec.os.name must be 'linux' or 'windows'"))
		}
	}

	// Проверяем контейнеры
	for i, c := range p.Spec.Containers {
		if err := c.Validate(i); err != nil {
			errs = append(errs, err...)
		}
	}

	return errs
}

func (c *Container) Validate(index int) []error {
	var errs []error

	// name
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, c.Name)
	if c.Name == "" {
		errs = append(errs, fmt.Errorf("container[%d].name is required", index))
	} else if !matched {
		errs = append(errs, fmt.Errorf("container[%d].name must be snake_case", index))
	}

	// image
	if c.Image == "" {
		errs = append(errs, fmt.Errorf("container[%d].image is required", index))
	} else {
		if !strings.HasPrefix(c.Image, "registry.bigbrother.io/") {
			errs = append(errs, fmt.Errorf("container[%d].image must be from registry.bigbrother.io", index))
		}
		if !strings.Contains(c.Image, ":") {
			errs = append(errs, fmt.Errorf("container[%d].image must contain tag", index))
		}
	}

	// ports
	if c.Ports != nil {
		if c.Ports.ContainerPort <= 0 || c.Ports.ContainerPort >= 65536 {
			errs = append(errs, fmt.Errorf("container[%d].ports.containerPort must be 1-65535", index))
		}
		if c.Ports.Protocol != "" && c.Ports.Protocol != "TCP" && c.Ports.Protocol != "UDP" {
			errs = append(errs, fmt.Errorf("container[%d].ports.protocol must be TCP or UDP", index))
		}
	}

	// probes
	if c.ReadinessProbe != nil {
		errs = append(errs, c.ReadinessProbe.Validate(index, "readinessProbe")...)
	}
	if c.LivenessProbe != nil {
		errs = append(errs, c.LivenessProbe.Validate(index, "livenessProbe")...)
	}

	// resources
	if len(c.Resources.Requests) == 0 && len(c.Resources.Limits) == 0 {
		errs = append(errs, fmt.Errorf("container[%d].resources is required", index))
	} else {
		errs = append(errs, c.Resources.Validate(index)...)
	}

	return errs
}

func (p *Probe) Validate(index int, probeType string) []error {
	var errs []error
	if p.HTTPGet.Path == "" {
		errs = append(errs, fmt.Errorf("container[%d].%s.httpGet.path is required", index, probeType))
	} else if !strings.HasPrefix(p.HTTPGet.Path, "/") {
		errs = append(errs, fmt.Errorf("container[%d].%s.httpGet.path must be absolute", index, probeType))
	}
	if p.HTTPGet.Port <= 0 || p.HTTPGet.Port >= 65536 {
		errs = append(errs, fmt.Errorf("container[%d].%s.httpGet.port must be 1-65535", index, probeType))
	}
	return errs
}

func (r *ResourceRequirements) Validate(index int) []error {
	var errs []error
	validateResourceMap := func(m map[string]string, field string) {
		for k, v := range m {
			switch k {
			case "cpu":
				if _, err := regexp.MatchString(`^\d+$`, v); err != nil {
					errs = append(errs, fmt.Errorf("container[%d].resources.%s.cpu must be integer", index, field))
				}
			case "memory":
				if !regexp.MustCompile(`^\d+(Gi|Mi|Ki)$`).MatchString(v) {
					errs = append(errs, fmt.Errorf("container[%d].resources.%s.memory must have units Gi, Mi or Ki", index, field))
				}
			default:
				errs = append(errs, fmt.Errorf("container[%d].resources.%s contains unknown key '%s'", index, field, k))
			}
		}
	}
	if r.Requests != nil {
		validateResourceMap(r.Requests, "requests")
	}
	if r.Limits != nil {
		validateResourceMap(r.Limits, "limits")
	}
	return errs
}

// ---------- Основная программа ----------

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: yamlvalid <path_to_yaml>")
		os.Exit(1)
	}

	path := os.Args[1]
	if !fileExists(path) {
		fmt.Printf("File not found: %s\n", path)
		os.Exit(1)
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var pod Pod
	if err := yaml.Unmarshal(content, &pod); err != nil {
		fmt.Printf("YAML decode error: %v\n", err)
		os.Exit(1)
	}

	errs := pod.Validate()
	if len(errs) > 0 {
		fmt.Println("Validation errors:")
		for _, e := range errs {
			fmt.Println("-", e)
		}
		os.Exit(1)
	}

	fmt.Println("YAML is valid")
}

func fileExists(path string) bool {
	info, err := os.Stat(filepath.Clean(path))
	return err == nil && !info.IsDir()
}
