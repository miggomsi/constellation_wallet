package main

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// monitorFileState will monitor the state of all files in .dag
// and act accordingly upon manipulation.
func (a *WalletApplication) monitorFileState() error {
	a.log.Info("Starting Watcher")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				} // If a JSONdata/*.json file is written to.
				if event.Op&fsnotify.Write&fsnotify.Create == fsnotify.Write|fsnotify.Create {
					a.log.Infof("modified file: %s", event.Name)
					switch fileModified := event.Name; {

					case fileModified == a.paths.LastTXFile:
						a.log.Debug("Last TX File has been modified")

					case fileModified == a.paths.KeyFile:
						a.log.Debug("Key File has been modified")
						a.RT.Events.Emit("wallet_keys", a.Wallet.PrivateKey.Key, a.Wallet.PublicKey.Key)

					case fileModified == "JSONdata/chart_data.json":
						a.log.Info("Chart Data file modified")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					a.sendError("", err)
					a.log.Error(err.Error())
				}
			}
		}
	}()

	err = watcher.Add(a.paths.DAGDir)
	if err != nil {
		a.sendError("Failed to start watcher. Reason: ", err)
		return err
	}
	return nil
}

func (a *WalletApplication) collectOSPath() error {
	user, err := user.Current()
	if err != nil {
		a.sendError("Unable to retrieve filesystem paths. Reason: ", err)
		a.log.Errorf("Unable to retrieve filesystem paths. Reason: ", err)
	}

	a.paths.HomeDir = user.HomeDir             // Home directory of the user
	a.paths.DAGDir = a.paths.HomeDir + "/.dag" // DAG directory for configuration files and wallet specific data
	a.paths.EncryptedDir = a.paths.DAGDir + "/encrypted_key"
	a.paths.KeyFile = a.paths.DAGDir + "/private_decrypted.pem" // DAG wallet keys
	a.paths.PubKeyFile = a.paths.EncryptedDir + "/pub.pem"
	a.paths.LastTXFile = a.paths.DAGDir + "/acct" // Account information

	a.log.Info("DAG Directory: " + a.paths.DAGDir)

	return nil
}

// This function is called by WailsInit and will initialize the dir structure.
func (a *WalletApplication) setupDirectoryStructure() error {
	err := os.MkdirAll(a.paths.DAGDir, os.ModePerm)
	if err != nil {
		return err
	}
	path := filepath.Join(a.paths.DAGDir, "txhistory.json")
	f, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY,
		0666,
	)
	defer f.Close()

	if !fileExists(path) {
		f.WriteString("{}") // initialies empty JSON object for frontend parsing
		f.Sync()
	}

	return nil
}

// writeToJSON is a helper function that will remove a requested file(filename),
// and recreate it with new data(data). This is to avoid ticking off the
// monitorFileState function with double write events.
func writeToJSON(filename string, data interface{}) error {
	user, err := user.Current()
	if err != nil {
		return err
	}
	JSON, err := json.Marshal(data)
	path := filepath.Join(user.HomeDir+"/.dag", filename)
	os.Remove(path)

	f, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_WRONLY,
		0666,
	)
	defer f.Close()

	f.Write(JSON)
	f.Sync()

	if err != nil {
		return err
	}
	return nil
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func reverseElement(elements []*txInformation) []*txInformation {
	reversed := []*txInformation{}
	for i := range elements {
		n := elements[len(elements)-1-i]
		reversed = append(reversed, n)
	}
	return reversed
}
