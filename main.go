package main

import (
	"flag"
	"net/http"

	"github.com/Strange-Account/go-mc-server-starter/config"
	"github.com/Strange-Account/go-mc-server-starter/packagetypes"

	log "github.com/sirupsen/logrus"
)

// Greeting screen
func greeting(name string) {
	log.Infof("ConfigFile: %s", "server-setup-config.yaml")
	log.Info(":::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
	log.Info("   Minecraft ServerStarter install/launcher in Go")
	log.Info("   (Created by A.Stranger)")
	log.Info(":::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
	log.Info("   This Go program will launch a Minecraft Forge/Fabric Modded server")
	log.Info()
	log.Infof("You are playing %s", name)
	log.Info("Starting to install/launch the server, lean back!")
	log.Info(":::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::::")
}

// Check inet connection
func checkConnection() bool {
	_, err := http.Get("http://clients3.google.com/generate_204")
	if err != nil {
		return false
	}
	return true
}

// Main function
func main() {
	// Define program flags
	configFileFlag := flag.String("c", "server-setup-config.yaml", "Path to server setup config yaml file")

	// Parse program flags
	flag.Parse()

	// Read server setup config
	myConfig := config.Read(*configFileFlag)

	// Create lockfile
	lockfile := config.NewLockFile()
	lockfile.Read(myConfig.Install.BaseInstallPath)

	// Check config version
	if myConfig.SpecVer < config.CURRENT_SPEC {
		log.Fatal("You are loading with an older Version of the specification!")
	}

	// Print greeting
	greeting(myConfig.Modpack.Name)

	// Check inet connection if needed
	if !checkConnection() && myConfig.Launch.CheckOffline {
		log.Fatal("Problems with the Internet connection, shutting down.")
	}

	// Get loader manager
	loaderManager := NewLoaderManager(myConfig.Launch, lockfile, myConfig.Install.BaseInstallPath)

	// Should we install pack and loader?
	if lockfile.CheckShouldInstall() {
		// Initiate package type
		// TODO: different package types / abstraction
		p := packagetypes.NewCursePack(myConfig)
		// Install package
		p.InstallPack()

		// Update lockfile
		lockfile.PackInstalled = true
		lockfile.PackUrl = myConfig.Install.ModpackUrl
		lockfile.Write(myConfig.Install.BaseInstallPath)

		// Install loader if needed
		if myConfig.Install.InstallLoader {
			forgeVersion := p.GetForgeVersion()
			mcVersion := p.GetMCVersion()
			loaderManager.installLoader(forgeVersion, mcVersion, myConfig.Install.InstallerArguments)
		}
	} else {
		log.Info("Server is already installed to correct version, to force install delete the serverstarter.lock File.")
	}

	// Start server handler
	loaderManager.handleServer()
}
