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
	"github.com/remeh/sizedwaitgroup"
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

		log.Info("Backup old files")
		backupOldFiles(p.basePath)

		downloadPack(p.basePath, url)

		log.Info("Processing Modpack")
		processModPack(p.basePath, p.config.Install.IgnoreFiles)

		log.Info("Processing manifest")
		mcVersion, forgeVersion, mods := processManifest(p.basePath, p.config.Install.FormatSpecific.IgnoreProject)

		if p.mcVersion == "" {
			p.mcVersion = mcVersion
		}

		if p.forgeVersion == "" {
			p.forgeVersion = forgeVersion
		}

		log.Info("Downloading mods")
		downloadMods(p.basePath, mods, p.config.Install.IgnoreFiles)
	}
}

func backupOldFiles(basePath string) {
	oldFilesPath := filepath.Join(basePath, "OLD_FILES_TO_DELETE")
	modsPath := filepath.Join(basePath, "mods")
	configPath := filepath.Join(basePath, "config")
	kubeJSPath := filepath.Join(basePath, "kubejs")

	err := os.RemoveAll(oldFilesPath)
	if err != nil {
		log.Error(err)
	}

	err = os.MkdirAll(oldFilesPath, os.ModePerm)
	if err != nil {
		log.Error(err)
	}

	err = os.Rename(modsPath, filepath.Join(oldFilesPath, "mods"))
	if err != nil {
		log.Error(err)
	}

	err = os.Rename(configPath, filepath.Join(oldFilesPath, "config"))
	if err != nil {
		log.Error(err)
	}

	err = os.Rename(kubeJSPath, filepath.Join(oldFilesPath, "kubejs"))
	if err != nil {
		log.Error(err)
	}
}

func downloadPack(basePath string, url string) {

	log.Infof("Attempting to download modpack Zip from %s.", url)
	modpackPath := filepath.Join(basePath, "modpack-download.zip")
	utils.DownloadFile(modpackPath, url)

	log.Infof("Unpacking modpack to %s", basePath)
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = utils.Unzip(modpackPath, absBasePath)
	if err != nil {
		log.Fatal(err)
	}
}

func processModPack(basePath string, ignoreFiles []string) {
	var globs []glob.Glob

	log.Info("Processing overrides")

	for _, ignoreFile := range ignoreFiles {
		globs = append(globs, glob.MustCompile(ignoreFile))
	}

	overridesPath := filepath.Join(basePath, "overrides")
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
}

func processManifest(basePath string, ignoreProjects []int) (mcVersion, forgeVersion string, mods []Files) {

	mods = []Files{}

	manifestFile := filepath.Join(basePath, "manifest.json")
	log.Infof("Reading manifest file: %s", manifestFile)
	jsonFile, err := os.Open(manifestFile)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	log.Info("Decoding manifest")
	manifest := Manifest{}
	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &manifest)

	mcVersion = manifest.Minecraft.Version
	if len(manifest.Minecraft.ModLoaders) > 0 {
		forgeVersion = manifest.Minecraft.ModLoaders[0].Id[6:]
	}

	for _, modFile := range manifest.Files {
		ok, _ := in_array(modFile.ProjectID, ignoreProjects)
		if !ok {
			log.Debug("Adding project %d - file %d to file list", modFile.ProjectID, modFile.FileID)
			mods = append(mods, modFile)
		} else {
			log.Debug("Skipping project %d - file %d", modFile.ProjectID, modFile.FileID)
		}
	}

	return mcVersion, forgeVersion, mods
}

func downloadMods(basePath string, mods []Files, ignoreFiles []string) {
	var downloadsUrls []string

	os.MkdirAll(filepath.Join(basePath, "mods"), os.ModePerm)

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

	ignorePatterns := []*regexp.Regexp{}
	for _, ignoreFiles := range ignoreFiles {
		if strings.HasPrefix(ignoreFiles, "mods/") {
			pattern, err := regexp.Compile(ignoreFiles[strings.LastIndex(ignoreFiles, "/")+1:])
			if err != nil {
				log.Fatal(err)
			}
			ignorePatterns = append(ignorePatterns, pattern)
		}
	}

	swg := sizedwaitgroup.New(5)
	for i, mod := range downloadsUrls {
		modName := path.Base(mod)

		ignored := false
		for _, ignorePattern := range ignorePatterns {
			if ignorePattern.MatchString(modName) {
				ignored = true
			}
		}

		if !ignored {
			log.Infof("(%d/%d) Loading mod %s ", i+1, len(mods), modName)
			swg.Add()
			go downloadSingleMod(basePath, mod, &swg)
		} else {
			log.Infof("(%d/%d) Skipped ignored mod: %s", i+1, len(mods), modName)
		}
	}

	swg.Wait()
}

func downloadSingleMod(basePath string, url string, swg *sizedwaitgroup.SizedWaitGroup) {
	defer swg.Done()

	modName := path.Base(url)
	destPath := filepath.Join(basePath, "mods", modName)
	err := utils.DownloadFile(destPath, url)
	if err != nil {
		log.Error(err)
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
