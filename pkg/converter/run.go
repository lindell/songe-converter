package converter

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CommandFlags Command Flags
type CommandFlags struct {
	KeepFiles bool
	DryRun    bool
	Quiet     bool
}

// Result Converted Hashes
type Result struct {
	Dir string

	OldHash string
	NewHash string
	Err     error
}

func makeLogger(flags CommandFlags) func(v interface{}) {
	return func(v interface{}) {
		if flags.Quiet == false {
			log.Print(v)
		}
	}
}

func Run(dir string, flags CommandFlags, c chan Result) {
	logger := makeLogger(flags)
	base := filepath.Base(dir)
	if base == "info.json" {
		dir = filepath.Dir(dir)
	}

	info := filepath.Join(dir, "info.json")
	infoJSON, infoErr := readInfo(info)
	if infoErr != nil && os.IsNotExist(infoErr) {
		logger("No info.json found in \"" + dir + "\", skipping!")

		result := Result{Dir: dir, OldHash: "", NewHash: "", Err: errors.New("info.json not found")}
		c <- result
		return
	} else if infoErr != nil {
		logger("Something went wrong when converting \"" + dir + "\"")
		logger(infoErr)

		result := Result{Dir: dir, OldHash: "", NewHash: "", Err: infoErr}
		c <- result
	} else {
		logger("Converting \"" + dir + "\"")
	}

	var newInfoJSON NewInfoJSON
	newInfoJSON.Version = "2.0.0"

	newInfoJSON.SongName = infoJSON.SongName
	newInfoJSON.SongSubName = ""
	newInfoJSON.LevelAuthorName = infoJSON.AuthorName
	newInfoJSON.SongAuthorName = infoJSON.SongSubName

	newInfoJSON.CustomData.Contributors = make([]Contributor, 0)
	for _, c := range infoJSON.Contributors {
		contributor := Contributor{Role: c.Role, Name: c.Name, IconPath: c.IconPath}
		newInfoJSON.CustomData.Contributors = append(newInfoJSON.CustomData.Contributors, contributor)
	}

	newInfoJSON.BeatsPerMinute = infoJSON.BeatsPerMinute
	newInfoJSON.SongTimeOffset = 0

	newInfoJSON.PreviewStartTime = infoJSON.PreviewStartTime
	newInfoJSON.PreviewDuration = infoJSON.PreviewDuration

	newInfoJSON.CoverImageFilename = infoJSON.CoverImagePath

	newInfoJSON.EnvironmentName = infoJSON.EnvironmentName
	newInfoJSON.CustomData.CustomEnvironment = infoJSON.CustomEnvironment
	newInfoJSON.CustomData.CustomEnvironmentHash = infoJSON.CustomEnvironmentHash

	toDelete := make([]string, 0)

	newInfoJSON.DifficultyBeatmapSets = make([]DifficultyBeatmapSet, 0)
	for _, diff := range infoJSON.DifficultyLevels {
		// Read JSON
		json := filepath.Join(dir, diff.JSONPath)
		toDelete = append(toDelete, json)

		diffJSON, diffErr := readDifficulty(json)
		if diffErr != nil && os.IsNotExist(diffErr) {
			logger(diff.JSONPath + " not found in \"" + dir + "\", skipping!")

			result := Result{Dir: dir, OldHash: "", NewHash: "", Err: errors.New(diff.JSONPath + " not found")}
			c <- result
			return
		} else if diffErr != nil {
			logger("Something went wrong when reading \"" + json + "\"")
			logger(diffErr)

			result := Result{Dir: dir, OldHash: "", NewHash: "", Err: diffErr}
			c <- result
		}

		// New File Name
		diffJSONFileName := strings.Replace(diff.JSONPath, ".json", ".dat", -1)

		var characteristic string
		if infoJSON.OneSaber {
			characteristic = "OneSaber"
		} else if diff.Characteristic == "One Saber" {
			characteristic = "OneSaber"
		} else if diff.Characteristic == "No Arrows" {
			characteristic = "NoArrows"
		} else if diff.Characteristic != "" {
			characteristic = diff.Characteristic
		} else {
			characteristic = "Standard"
		}

		var beatmapSet DifficultyBeatmapSet
		beatmapSetIdx := -1
		for i := range newInfoJSON.DifficultyBeatmapSets {
			if newInfoJSON.DifficultyBeatmapSets[i].BeatmapCharacteristicName == characteristic {
				beatmapSet = newInfoJSON.DifficultyBeatmapSets[i]
				beatmapSetIdx = i
				break
			}
		}

		if beatmapSetIdx == -1 {
			beatmapSet.BeatmapCharacteristicName = characteristic
			beatmapSet.DifficultyBeatmaps = make([]DifficultyBeatmap, 0)

			newInfoJSON.DifficultyBeatmapSets = append(newInfoJSON.DifficultyBeatmapSets, beatmapSet)
			beatmapSetIdx = len(newInfoJSON.DifficultyBeatmapSets) - 1
		}

		var difficulty DifficultyBeatmap
		difficulty.Difficulty = diff.Difficulty
		difficulty.DifficultyRank = getRank(diff.Difficulty)
		difficulty.CustomData.DifficultyLabel = diff.DifficultyLabel
		difficulty.BeatmapFilename = diffJSONFileName
		difficulty.NoteJumpMovementSpeed = diffJSON.NoteJumpSpeed
		difficulty.NoteJumpStartBeatOffset = diffJSON.NoteJumpStartBeatOffset
		difficulty.CustomData.EditorOffset = diff.Offset
		difficulty.CustomData.EditorOldOffset = diff.OldOffset
		difficulty.CustomData.Warnings = diffJSON.Warnings
		difficulty.CustomData.Information = diffJSON.Information
		difficulty.CustomData.Suggestions = diffJSON.Suggestions
		difficulty.CustomData.Requirements = diffJSON.Requirements

		if difficulty.CustomData.Warnings == nil {
			difficulty.CustomData.Warnings = make([]string, 0)
		}

		if difficulty.CustomData.Information == nil {
			difficulty.CustomData.Information = make([]string, 0)
		}

		if difficulty.CustomData.Suggestions == nil {
			difficulty.CustomData.Suggestions = make([]string, 0)
		}

		if difficulty.CustomData.Requirements == nil {
			difficulty.CustomData.Requirements = make([]string, 0)
		}

		needsMapExt := checkForMapExt(&diffJSON)
		hasMapExt := stringInSlice("Mapping Extensions", difficulty.CustomData.Requirements)
		if needsMapExt == true && hasMapExt == false {
			difficulty.CustomData.Requirements = append(difficulty.CustomData.Requirements, "Mapping Extensions")
		}

		difficulty.CustomData.ColorLeft = diffJSON.ColorLeft
		difficulty.CustomData.ColorRight = diffJSON.ColorRight

		newInfoJSON.Shuffle = diffJSON.Shuffle
		newInfoJSON.ShufflePeriod = diffJSON.ShufflePeriod
		newInfoJSON.SongFilename = diff.AudioPath

		if diffJSON.BeatsPerMinute != 0 {
			newInfoJSON.BeatsPerMinute = diffJSON.BeatsPerMinute
		}

		var newDiffJSON NewDifficultyJSON
		newDiffJSON.Version = "2.0.0"

		// Set
		newDiffJSON.BPMChanges = diffJSON.BPMChanges
		newDiffJSON.Events = diffJSON.Events
		newDiffJSON.Notes = diffJSON.Notes
		newDiffJSON.Obstacles = diffJSON.Obstacles
		newDiffJSON.Bookmarks = diffJSON.Bookmarks

		if newDiffJSON.BPMChanges == nil {
			newDiffJSON.BPMChanges = make([]BPMChange, 0)
		}

		if newDiffJSON.Events == nil {
			newDiffJSON.Events = make([]Event, 0)
		}

		if newDiffJSON.Notes == nil {
			newDiffJSON.Notes = make([]Note, 0)
		}

		if newDiffJSON.Obstacles == nil {
			newDiffJSON.Obstacles = make([]Obstacle, 0)
		}

		if newDiffJSON.Bookmarks == nil {
			newDiffJSON.Bookmarks = make([]Bookmark, 0)
		}

		// Save
		diffJSONBytes, _ := JSONMarshal(newDiffJSON)
		difficulty.Bytes = diffJSONBytes
		beatmapSet.DifficultyBeatmaps = append(beatmapSet.DifficultyBeatmaps, difficulty)
		newInfoJSON.DifficultyBeatmapSets[beatmapSetIdx] = beatmapSet

		diffJSONPath := filepath.Join(dir, diffJSONFileName)
		if flags.DryRun == false {
			_ = ioutil.WriteFile(diffJSONPath, diffJSONBytes, 0644)
		}
	}

	for _, set := range newInfoJSON.DifficultyBeatmapSets {
		sort.Slice(set.DifficultyBeatmaps, func(i, j int) bool {
			return set.DifficultyBeatmaps[i].DifficultyRank < set.DifficultyBeatmaps[j].DifficultyRank
		})
	}

	allBytes := make([]byte, 0)
	infoJSONBytes, _ := JSONMarshalPretty(newInfoJSON)

	allBytes = append(allBytes, infoJSONBytes...)
	for _, set := range newInfoJSON.DifficultyBeatmapSets {
		for _, d := range set.DifficultyBeatmaps {
			allBytes = append(allBytes, d.Bytes...)
		}
	}

	infoJSONPath := filepath.Join(dir, "info.dat")
	if flags.DryRun == false {
		_ = ioutil.WriteFile(infoJSONPath, infoJSONBytes, 0644)
	}

	oldHash, err := calculateOldHash(infoJSON, dir)
	if err != nil {
		logger("Something went wrong when converting \"" + dir + "\"")
		logger(err)

		result := Result{Dir: dir, OldHash: "", NewHash: "", Err: err}
		c <- result
	}

	if flags.KeepFiles == false && flags.DryRun == false {
		err := os.Remove(info)
		if err != nil {
			logger("Failed to delete \"" + info + "\"")
		}

		for _, d := range toDelete {
			err := os.Remove(d)
			if err != nil {
				logger("Failed to delete \"" + d + "\"")
			}
		}
	}

	newHash := calculateHashSHA1(allBytes)
	result := Result{Dir: dir, OldHash: oldHash, NewHash: newHash, Err: nil}
	c <- result
}
