package packagetypes

type Manifest struct {
	Minecraft       Minecraft `json:"minecraft"`
	ManifestType    string    `json:"manifestType"`
	ManifestVersion int       `json:"manifestVersion"`
	Name            string    `json:"name": "ATM6 to the Sky"`
	Version         string    `json:"version": "1.0.5"`
	Author          string    `json:"author": "White_Phant0m & oly2o6"`
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
