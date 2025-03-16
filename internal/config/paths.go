package config

// outsource to a common package as this is used across several blindbit programs
import "github.com/setavenger/blindbitd/src/utils"

var (
	DirectoryPath = "~/.blindbit-scan"

	PathLogs     string
	PathConfig   string
	PathDbWallet string
	PathDbNWC    string
)

// needed for the flag default
const DefaultDirectoryPath = "~/.blindbit-scan"

const dataPath = "/data"
const PathEndingConfig = "/blindbit.toml"
const PathEndingWallet = dataPath + "/wallet"
const PathEndingNWC = dataPath + "/nwc"
const PathEndingKeys = dataPath + "/keys"

func SetPaths(baseDirectory string) {
	if baseDirectory != "" {
		DirectoryPath = baseDirectory
	}

	// do this so that we can parse ~ in paths
	DirectoryPath = utils.ResolvePath(DirectoryPath)

	// PathLogs = DirectoryPath + logsPath

	PathConfig = DirectoryPath + PathEndingConfig
	PathDbWallet = DirectoryPath + PathEndingWallet
	PathDbNWC = DirectoryPath + PathEndingNWC

	// create the directories
	utils.TryCreateDirectoryPanic(DirectoryPath)

	utils.TryCreateDirectoryPanic(DirectoryPath + dataPath)
	// utils.TryCreateDirectoryPanic(PathDbNWC)
}
