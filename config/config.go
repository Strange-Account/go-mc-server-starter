package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

const CURRENT_SPEC = 2

type ConfigFile struct {
	SpecVer int64         `yaml:"_specver"`
	Modpack ModpackConfig `yaml:"modpack"`
	Install InstallConfig `yaml:"install"`
	Launch  LaunchConfig  `yaml:"launch"`
}

type ModpackConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type InstallConfig struct {
	MCVersion          string                 `yaml:"mcVersion"`
	LoaderVersion      string                 `yaml:"loaderVersion"`
	InstallerUrl       string                 `yaml:"installerUrl"`
	InstallerArguments []string               `yaml:"installerArguments"`
	ModpackUrl         string                 `yaml:"modpackUrl"`
	ModpackFormat      string                 `yaml:"modpackFormat"`
	FormatSpecific     FormatSpecificConfig   `yaml:"formatSpecific"`
	BaseInstallPath    string                 `yaml:"baseInstallPath"`
	IgnoreFiles        []string               `yaml:"ignoreFiles"`
	AdditionalFiles    []AdditionalFileConfig `yaml:"additionalFiles"`
	LocalFiles         []LocalFileConfig      `yaml:"localFiles"`
	CheckFolder        bool                   `yaml:"checkFolder"`
	InstallLoader      bool                   `yaml:"installLoader"`
}

type LaunchConfig struct {
	Spongefix      bool     `yaml:"spongefix"`
	RamDisk        bool     `yaml:"ramDisk"`
	CheckOffline   bool     `yaml:"checkOffline"`
	MaxRam         string   `yaml:"maxRam"`
	AutoRestart    bool     `yaml:"autoRestart"`
	CrashLimit     int      `yaml:"crashLimit"`
	CrashTimer     string   `yaml:"crashTimer"`
	PreJavaArgs    string   `yaml:"preJavaArgs"`
	StartFile      string   `yaml:"startFile"`
	ForcedJavaPath string   `yaml:"forcedJavaPath"`
	JavaArgs       []string `yaml:"javaArgs"`
}

type AdditionalFileConfig struct {
	Url         string `yaml:"url"`
	Destination string `yaml:"destination"`
}

type LocalFileConfig struct {
	Url         string `yaml:"url"`
	Destination string `yaml:"destination"`
}

type FormatSpecificConfig struct {
	IgnoreProject []int `yaml:"ignoreProject"`
}

func Read(path string) *ConfigFile {

	c := &ConfigFile{}

	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("yamlFile.Get err #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
