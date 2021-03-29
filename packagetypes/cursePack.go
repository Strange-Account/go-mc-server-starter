package packagetypes

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Strange-Account/go-mc-server-starter/config"
	"github.com/Strange-Account/go-mc-server-starter/utils"
	"github.com/gobwas/glob"
	log "github.com/sirupsen/logrus"
)

// type packageType interface {
// 	installPack()
// 	getForgeVersion() string
// 	getMCVersion() string
// }

func NewCursePack(config *config.ConfigFile) *cursePackType {
	p := cursePackType{}

	p.config = config
	p.forgeVersion = config.Install.LoaderVersion
	p.mcVersion = config.Install.MCVersion
	p.basePath = config.Install.BaseInstallPath

	return &p
}

type cursePackType struct {
	config       *config.ConfigFile
	forgeVersion string
	mcVersion    string
	basePath     string
}

func (p *cursePackType) GetForgeVersion() string {
	return p.forgeVersion
}

func (p *cursePackType) GetMCVersion() string {
	return p.mcVersion
}

func (p *cursePackType) InstallPack() {
	if p.config.Install.ModpackUrl != "" {
		url := p.config.Install.ModpackUrl
		os.MkdirAll(p.basePath, os.ModePerm)

		modsPath := filepath.Join(p.basePath, "mods")
		oldModsPath := filepath.Join(p.basePath, "old-mods")

		if _, err := os.Stat(modsPath); !os.IsNotExist(err) {
			if _, err := os.Stat(oldModsPath); !os.IsNotExist(err) {
				err := os.RemoveAll(oldModsPath)
				if err != nil {
					log.Fatal(err)
				}
			}
			err := os.Rename(modsPath, oldModsPath)
			if err != nil {
				log.Fatal(err)
			}
		}
		err := os.MkdirAll(modsPath, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Attempting to download modpack Zip.")

		modpackPath := filepath.Join(p.basePath, "modpack-download.zip")
		utils.DownloadFile(modpackPath, url)

		absModpackPath, err := filepath.Abs(modpackPath)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Downloaded the modpack zip file to %s", absModpackPath)

		log.Debugf("Unpacking modpack to %s", p.basePath)
		absBasePath, err := filepath.Abs(p.basePath)
		if err != nil {
			log.Fatal(err)
		}
		utils.Unzip(absModpackPath, absBasePath)

		p.postProcess()
	}
}

func (p *cursePackType) postProcess() {
	mods := []Files{}
	var globs []glob.Glob

	log.Info("Post-processing overrides")

	for _, ignoreFile := range p.config.Install.IgnoreFiles {
		globs = append(globs, glob.MustCompile(ignoreFile))
	}

	overridesPath := filepath.Join(p.config.Install.BaseInstallPath, "overrides")
	err := filepath.Walk(overridesPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == "overrides" || info.IsDir() {
				return nil
			}
			dest := path[10:]
			for _, g := range globs {
				if g.Match(dest) {
					log.Infof("Skipping file: %s", dest)
					return nil
				}
			}
			log.Infof("Moving file: %s", dest)
			destFolder := filepath.Dir(dest)
			err = os.MkdirAll(destFolder, os.ModePerm)
			if err != nil {
				return err
			}
			err = os.Rename(path, dest)
			return err
		})
	if err != nil {
		log.Error(err)
	}

	log.Info("Remove override directory")
	os.RemoveAll(overridesPath)

	log.Infof("Reading manifest file: %s", filepath.Join(p.basePath, "manifest.json"))
	jsonFile, err := os.Open(filepath.Join(p.basePath, "manifest.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	log.Info("Decoding manifest")
	manifest := Manifest{}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &manifest)

	if p.mcVersion == "" {
		p.mcVersion = manifest.Minecraft.Version
	}

	if p.forgeVersion == "" && len(manifest.Minecraft.ModLoaders) > 0 {
		p.forgeVersion = manifest.Minecraft.ModLoaders[0].Id[6:]
	}

	for _, modFile := range manifest.Files {
		ok, _ := in_array(modFile.ProjectID, p.config.Install.FormatSpecific.IgnoreProject)
		if !ok {
			log.Debug("Adding project %d - file %d to file list", modFile.ProjectID, modFile.FileID)
			mods = append(mods, modFile)
		} else {
			log.Debug("Skipping project %d - file %d", modFile.ProjectID, modFile.FileID)
		}
	}

	p.downloadMods(mods)
}

func (p *cursePackType) downloadMods(mods []Files) {
	var downloadsUrls []string

	for _, modFile := range mods {
		url := "https://cursemeta.dries007.net/" +
			strconv.Itoa(modFile.ProjectID) + "/" + strconv.Itoa(modFile.FileID) + ".json"
		// log.Printf("Download url is: %s\n", url)

		spaceClient := http.Client{
			Timeout: time.Second * 2, // Timeout after 2 seconds
		}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Fatal(err)
		}
		res, getErr := spaceClient.Do(req)
		if getErr != nil {
			log.Fatal(getErr)
		}

		if res.Body != nil {
			defer res.Body.Close()
		}
		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}
		var result map[string]interface{}
		json.Unmarshal([]byte(body), &result)
		downloadsUrls = append(downloadsUrls, result["DownloadURL"].(string))
	}

	p.processMods(downloadsUrls)
}

func (p *cursePackType) processMods(mods []string) {
	ignorePatterns := []*regexp.Regexp{}
	for _, ignoreFiles := range p.config.Install.IgnoreFiles {
		if strings.HasPrefix(ignoreFiles, "mods/") {
			pattern, err := regexp.Compile(ignoreFiles[strings.LastIndex(ignoreFiles, "/")+1:])
			if err != nil {
				log.Fatal(err)
			}
			ignorePatterns = append(ignorePatterns, pattern)
		}
	}

	for i, mod := range mods {
		modName := path.Base(mod)
		for _, ignorePattern := range ignorePatterns {
			if ignorePattern.MatchString(modName) {
				log.Infof("(%d/%d) Skipped ignored mod: %s", i, len(mods)-1, modName)
			} else {
				destPath := filepath.Join(p.config.Install.BaseInstallPath, "mods", modName)
				log.Infof("(%d/%d) Loading mod %s ", i, len(mods)-1, modName)
				err := utils.DownloadFile(destPath, mod)
				if err != nil {
					log.Error(err)
				}
			}
		}

	}
}

func in_array(val int, array []int) (ok bool, i int) {
	for i = range array {
		if val == array[i] {
			return true, i
		}
	}
	return false, -1
}
