package converter

import (
	"encoding/json"
	"errors"

	"github.com/TomOnTime/utfutil"
)

// OldInfoJSON is the old beatmap info file
type OldInfoJSON struct {
	SongName    string `json:"songName"`
	SongSubName string `json:"songSubName"`
	AuthorName  string `json:"authorName"`

	Contributors []struct {
		Role     string `json:"role"`
		Name     string `json:"name"`
		IconPath string `json:"iconPath"`
	} `json:"contributors"`

	BeatsPerMinute        float64 `json:"beatsPerMinute"`
	PreviewStartTime      float64 `json:"previewStartTime"`
	PreviewDuration       float64 `json:"previewDuration"`
	CoverImagePath        string  `json:"coverImagePath"`
	EnvironmentName       string  `json:"environmentName"`
	OneSaber              bool    `json:"oneSaber"`
	CustomEnvironment     string  `json:"customEnvironment"`
	CustomEnvironmentHash string  `json:"customEnvironmentHash"`

	DifficultyLevels []struct {
		Difficulty      string `json:"difficulty"`
		DifficultyRank  int    `json:"difficultyRank"`
		AudioPath       string `json:"audioPath"`
		JSONPath        string `json:"jsonPath"`
		Offset          int    `json:"offset"`
		OldOffset       int    `json:"oldOffset"`
		ChromaToggle    string `json:"chromaToggle"`
		CustomColors    bool   `json:"customColors"`
		Characteristic  string `json:"characteristic"`
		DifficultyLabel string `json:"difficultyLabel"`
	} `json:"difficultyLevels"`
}

func readInfo(path string) (OldInfoJSON, error) {
	bytes, err := utfutil.ReadFile(path, utfutil.UTF8)
	if err != nil {
		return OldInfoJSON{}, err
	}

	valid := IsJSON(bytes)
	if valid == false {
		invalidError := errors.New("Invalid info.json")
		return OldInfoJSON{}, invalidError
	}

	var infoJSON OldInfoJSON
	json.Unmarshal(bytes, &infoJSON)

	return infoJSON, nil
}
