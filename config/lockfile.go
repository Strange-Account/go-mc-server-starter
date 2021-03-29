package config

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type LockFile struct {
	LoaderInstalled    bool   `yaml:"loaderInstalled"`
	PackInstalled      bool   `yaml:"packInstalled"`
	LoaderVersion      string `yaml:"loaderVersion"`
	McVersion          string `yaml:"mcVersion"`
	PackUrl            string `yaml:"packUrl"`
	SpongeBootstrapper string `yaml:"spongeBootstrapper"`
}

func NewLockFile() *LockFile {
	l := LockFile{}
	l.LoaderInstalled = false
	l.PackInstalled = false
	l.LoaderVersion = ""
	l.McVersion = ""
	l.PackUrl = ""
	l.SpongeBootstrapper = ""

	return &l
}

func (l *LockFile) Read(basePath string) error {
	lockFile := filepath.Join(basePath, "serverstarter.lock")

	if _, err := os.Stat(lockFile); err == nil {
		f, err := os.Open(lockFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		d := yaml.NewDecoder(f)
		if err := d.Decode(l); err != nil {
			return err
		}
	}

	return nil
}

func (l *LockFile) Write(basePath string) error {
	lockFile := filepath.Join(basePath, "serverstarter.lock")

	f, err := os.Create(lockFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	e := yaml.NewEncoder(f)
	if err := e.Encode(l); err != nil {
		return err
	}

	return nil
}

func (l *LockFile) CheckShouldInstall() bool {
	return !l.LoaderInstalled || !l.PackInstalled
}
