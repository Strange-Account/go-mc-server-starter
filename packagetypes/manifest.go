package packagetypes

type Manifest struct {
	Minecraft       Minecraft `json:"minecraft"`
	ManifestType    string    `json:"manifestType"`
	ManifestVersion int       `json:"manifestVersion"`
	Name            string    `json:"name"`
	Version         string    `json:"version"`
	Author          string    `json:"author"`
	Files           []Files   `json:"files"`
}

type Minecraft struct {
	Version    string       `json:"version"`
	ModLoaders []ModLoaders `json:"modLoaders"`
}

type ModLoaders struct {
	Id      string `json:"id"`
	Primary bool   `json:"primary"`
}

type Files struct {
	ProjectID int  `json:"projectID"`
	FileID    int  `json:"fileID"`
	Required  bool `json:"required"`
}
