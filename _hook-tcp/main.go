package main

import (
	"fmt"
	"github.com/peakedshout/go-CFC/loger"
	"os"
	"os/exec"
)

func main() {
	checkFile()
	makeConfig()
	makeAllBin()
}
func errCheck(err error) {
	if err != nil {
		loger.SetLogError(err)
	}
}
func checkFile() {
	var err error
	_, err = os.Stat("./client/client.go")
	errCheck(err)
	_, err = os.Stat("./server/server.go")
	errCheck(err)
	_, err = os.Stat("./config/configCN_M.json")
	errCheck(err)
	_, err = os.Stat("./config/configEN_M.json")
	errCheck(err)
	for key, value := range goGoArchMap {
		for _, one := range value {
			err = os.MkdirAll("./asset/"+key+"-"+one, 0777)
			errCheck(err)
		}
	}
}

var goGoosMap = map[string]string{
	"win":   "windows",
	"mac":   "darwin",
	"linux": "linux",
}
var goGoArchMap = map[string][]string{
	"linux": {"386", "amd64", "arm64", "arm"},
	"mac":   {"amd64", "arm64"},
	"win":   {"386", "amd64", "arm64", "arm"},
}

var goEnvMap = map[string]string{
	"GOPATH":      os.Getenv("GOPATH"),
	"GOROOT":      os.Getenv("GOROOT"),
	"GO111MODULE": os.Getenv("GO111MODULE"),
	"_":           os.Getenv("_"),
	"PWD":         os.Getenv("PWD"),
	"HOME":        os.Getenv("HOME"),
	"PATH":        os.Getenv("PATH"),
}

func cleanGoEnv() {
	var err error
	os.Clearenv()
	for key, value := range goEnvMap {
		err = os.Setenv(key, value)
		errCheck(err)
	}
}

func makeAllBin() {
	for key, value := range goGoArchMap {
		for _, one := range value {
			makeBin(key, one)
		}
	}
}

func makeBin(goos, arch string) {
	var err error
	var cmd *exec.Cmd
	var b []byte
	cleanGoEnv()
	err = os.Setenv("CGO_ENABLED", "0")
	errCheck(err)
	err = os.Setenv("GOOS", goGoosMap[goos])
	errCheck(err)
	err = os.Setenv("GOARCH", arch)

	errCheck(err)
	s := getOutPath(goos, arch, true)
	cmd = exec.Command("go", "build", "-ldflags", "-s -w", "-o", s, "./server/server.go")
	cmd.Env = append(os.Environ())
	b, err = cmd.CombinedOutput()
	fmt.Println("s", goos, arch, string(b), err)
	//err = cmd.Run()
	errCheck(err)
	err = os.Chmod(s, 0777)
	errCheck(err)

	c := getOutPath(goos, arch, false)
	cmd = exec.Command("go", "build", "-ldflags", "-s -w", "-o", c, "./client/client.go")
	cmd.Env = append(os.Environ())
	b, err = cmd.CombinedOutput()
	fmt.Println("c", goos, arch, string(b), err)
	err = os.Chmod(c, 0777)
	errCheck(err)
}

func getOutPath(goos, arch string, isServer bool) string {
	out := ""
	if isServer {
		out = "./asset/" + goos + "-" + arch + "/cfc_hook_server_" + goos + "-" + arch
	} else {
		out = "./asset/" + goos + "-" + arch + "/cfc_hook_client_" + goos + "-" + arch
	}
	if goos == "win" || goos == "windows" {
		out += ".exe"
	}
	return out
}

var configMMap = map[string]string{
	"CN": "CN_M",
	"EN": "EN_M",
	"":   "",
}

func makeConfig() {
	fn1 := func(key string) string {
		return "./config/config" + key + ".json"
	}
	fn2 := func(key string) string {
		return "./asset/config" + key
	}
	for key, value := range configMMap {
		b, err := os.ReadFile(fn1(value))
		errCheck(err)
		err = os.MkdirAll(fn2(key), 0777)
		errCheck(err)
		err = os.WriteFile(fn2(key)+"/config.json", b, 0777)
		errCheck(err)
	}
}
