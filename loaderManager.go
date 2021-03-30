package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Strange-Account/go-mc-server-starter/config"
	"github.com/Strange-Account/go-mc-server-starter/utils"
)

type loaderManager struct {
	launchConfig config.LaunchConfig
	lockfile     *config.LockFile
	basePath     string
}

func NewLoaderManager(config config.LaunchConfig, lockfile *config.LockFile, basePath string) *loaderManager {
	l := loaderManager{}
	l.launchConfig = config
	l.lockfile = lockfile
	l.basePath = basePath

	return &l
}

func (l *loaderManager) installLoader(loaderVersion string, mcVersion string, installerArguments []string) {
	url := "http://files.minecraftforge.net/maven/net/minecraftforge/forge/{{@mcversion@}}-{{@loaderversion@}}/forge-{{@mcversion@}}-{{@loaderversion@}}-installer.jar"
	url = strings.ReplaceAll(url, "{{@loaderversion@}}", loaderVersion)
	url = strings.ReplaceAll(url, "{{@mcversion@}}", mcVersion)

	installerPath := filepath.Join(l.basePath, "installer.jar")

	log.Infof("Attempting to download installer from %s", url)
	err := utils.DownloadFile(installerPath, url)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Starting installation of Loader, installer output incoming")
	log.Info("Check log for installer for more information")

	absInstallerPath, err := filepath.Abs(installerPath)
	if err != nil {
		log.Fatal(err)
	}

	// cmd_prep := "java -jar " + absInstallerPath
	var args []string
	args = append(args, "-jar", absInstallerPath)
	args = append(args, installerArguments...)
	cmd := exec.Command("java", args...)
	cmd.Dir = l.basePath

	var stdBuffer bytes.Buffer
	mw := io.MultiWriter(os.Stdout, &stdBuffer)

	cmd.Stdout = mw
	cmd.Stderr = mw

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	log.Info(stdBuffer.String())

	log.Info("Done installing loader, deleting installer!")

	l.lockfile.LoaderInstalled = true
	l.lockfile.LoaderVersion = loaderVersion
	l.lockfile.McVersion = mcVersion
	l.lockfile.Write(l.basePath)

	err = os.Remove(absInstallerPath)
	if err != nil {
		log.Fatal(err)
	}

	l.checkEULA()
}

func (l *loaderManager) checkEULA() {
	eulaFile := filepath.Join(l.basePath, "eula.txt")
	var lines []string

	if _, err := os.Stat(eulaFile); err == nil {
		f, err := os.Open(eulaFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
		f.Close()
	} else if os.IsNotExist(err) {
		f, err := os.Create(eulaFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		lines = append(lines, "#By changing the setting below to TRUE you are indicating your agreement to our EULA (https://account.mojang.com/documents/minecraft_eula).")
		lines = append(lines, "#"+time.Now().Format("E MMM d HH:mm:ss O y"))
		lines = append(lines, "eula=false")

		w := bufio.NewWriter(f)
		for _, line := range lines {
			fmt.Fprintln(w, line)
		}

		err = w.Flush()
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	} else {
		log.Fatal(err)
	}

	if len(lines) > 2 && !strings.Contains(lines[2], "true") {
		log.Info("You have not accepted the eula yet.")
		log.Info("By typing TRUE you are indicating your agreement to the EULA of Mojang.")
		log.Info("Read it at https://account.mojang.com/documents/minecraft_eula before accepting it.")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		if strings.Contains(strings.ToLower(text), "true") {
			log.Info("You have accepted the EULA.")
			lines[2] = "eula=true"

			f, err := os.Create(eulaFile)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			w := bufio.NewWriter(f)
			for _, line := range lines {
				fmt.Fprintln(w, line)
			}

			err = w.Flush()
			if err != nil {
				log.Fatal(err)
			}
			f.Close()
		}
	}
}

func (l *loaderManager) handleServer() {
	var startTimes []time.Time
	counter := 0

	crashLimit := l.launchConfig.CrashLimit
	crashTimer, err := time.ParseDuration(l.launchConfig.CrashTimer)
	if err != nil {
		log.Fatal(err)
	}

	for len(startTimes) < crashLimit {
		startTimes = append(startTimes, time.Now())
		counter++

		log.Infof("Starting server. Try %d", counter)
		l.startServer()

		if !l.launchConfig.AutoRestart {
			return
		}

		var tempStartTimes []time.Time
		for _, startTime := range startTimes {
			if startTime.Add(crashTimer).After(time.Now()) {
				tempStartTimes = append(tempStartTimes, startTime)
			}
		}
		startTimes = tempStartTimes
		log.Errorf("Server crashed %d times in the last %s.", len(startTimes), l.launchConfig.CrashTimer)
		log.Info("Restarting in 10 Seconds")
		log.Info("Press Ctrl+C to cancel.")
		time.Sleep(10 * time.Second)
	}
}

func (l *loaderManager) startServer() {
	// Process os signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// basePath := l.config.Install.BaseInstallPath

	// World folder
	// level := "world"

	// Get level folder
	// server properties
	propertiesFile := filepath.Join(l.basePath, "server.properties")
	if _, err := os.Stat(propertiesFile); err == nil {
		f, err := os.Open(propertiesFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}

	var startFile string
	if l.launchConfig.Spongefix {
		startFile = ""
	} else {
		startFile = strings.ReplaceAll(l.launchConfig.StartFile, "{{@mcversion@}}", l.lockfile.McVersion)
		startFile = strings.ReplaceAll(startFile, "{{@loaderversion@}}", l.lockfile.LoaderVersion)
	}
	log.Infof("Using launcher file: %s", startFile)

	launchJar := filepath.Join(l.basePath, startFile)
	launchJar, err := filepath.Abs(launchJar)
	if err != nil {
		log.Error(err)
	}
	java := "java"

	if l.launchConfig.ForcedJavaPath != "" {
		java = l.launchConfig.ForcedJavaPath
	}

	log.Info("Starting Loader, output incoming")
	log.Info("For output of this check the server log")

	// Build start command
	var args []string
	args = append(args, l.launchConfig.JavaArgs...)
	args = append(args, "-jar", launchJar)
	cmd := exec.Command(java, args...)
	cmd.Dir = l.basePath

	// Redirect Stdout/Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	log.Debug(cmd)

	// Start Server
	err = cmd.Start()
	if err != nil {
		log.Error(err)
	}

	go func(cmd *exec.Cmd) {
		sig := <-signals
		log.Infof("Recieved signal %s", sig)
		log.Info("Stopping server and wait 10 seconds to complete")
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(10 * time.Second)
		os.Exit(0)
	}(cmd)

	err = cmd.Wait()
	if err != nil {
		log.Error(err)
	}
}
