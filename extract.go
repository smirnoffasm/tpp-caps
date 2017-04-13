package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/glide/path"
	"github.com/apcera/termtables"
)

var providers = make(map[string]string)

func walkerFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Print(err)
		return nil
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "tpp-providers" {
			//if len(parts) == i+3 {
			//	fmt.Println(path)
			//}
			if len(parts) == i+3 && (parts[i+1]+".go" == parts[i+2]) {
				providers[parts[i+1]] = path
			} else if len(parts) == i+3 && (parts[i+2] == "search_init.go") {
				providers[parts[i+1]] = path
			}
		}
	}
	return nil
}

var tppPaths = []string{}

func locateTppProviders(path string, info os.FileInfo, err error) error {
	if err != nil {
		panic(err)
	}
	parts := strings.Split(path, "/")
	if info.IsDir() && parts[len(parts)-1] == "tpp-providers" {
		inSrc := false
		for _, p := range parts {
			if p == "src" {
				inSrc = true
				break
			}
		}
		if inSrc {
			tppPaths = append(tppPaths, path)
		}
	}
	return nil
}

func main() {
	tppDir := ""
	if (len(os.Args) > 1 && os.Args[1] == "help") || len(os.Args) > 2 {
		usage()
		return
	}
	if len(os.Args) == 2 {
		tppDir = os.Args[1]
		fi, err := os.Stat(tppDir)
		if os.IsNotExist(err) {
			log.Fatalln("directory", tppDir, "does not exist")
		}
		if !fi.IsDir() {
			log.Fatalln(tppDir, "is not a directory")
		}
	} else {
		for _, gopath := range path.Gopaths() {
			filepath.Walk(gopath, locateTppProviders)
		}
		if len(tppPaths) == 0 {
			log.Println("can not find package `tpp-providers` in GOPATH. If it exist try to set path manualy")
			usage()
			return
		}
		if len(tppPaths) > 1 {
			log.Println("found several `tpp-providers` in GOPATH. Specify one you interested in")
			log.Println(tppPaths)
			usage()
			return
		}
		tppDir = tppPaths[0]
	}

	filepath.Walk(tppDir, walkerFunc)
	if len(providers) == 0 {
		return
	}

	table := termtables.CreateTable()
	table.AddTitle("Provider capabilities")

	for pr, initPath := range providers {
		c, err := ioutil.ReadFile(initPath)
		if err != nil {
			c, err = ioutil.ReadFile(filepath.Join(tppDir, pr, "search_init.go"))
			if err != nil {
				log.Fatal(err)
			}
		}
		cstr := string(c)
		hotelAvStart := strings.Index(cstr, "HotelAv.AddCapabilities")
		regionAvStart := strings.Index(cstr, "RegionAv.AddCapabilities")
		if hotelAvStart == -1 && regionAvStart == -1 {
			log.Println("[WARN] could not read <", pr, "> hotel capabilities")
			continue
		}

		capabilities := []string{}

		if hotelAvStart != -1 {
			capsString := readCapabilityString(cstr, hotelAvStart, pr)
			capabilities = append(capabilities, capList(capsString)...)
		}
		if regionAvStart != -1 {
			reqCapsString := readCapabilityString(cstr, regionAvStart, pr)
			for _, regCap := range capList(reqCapsString) {
				alreadyExist := false
				for _, htlCap := range capabilities {
					if htlCap == regCap {
						alreadyExist = true
						break
					}
				}
				if !alreadyExist {
					capabilities = append(capabilities, regCap)
				}
			}
		}
		if len(capabilities) == 0 {
			log.Println("[WARN] could not read <", pr, "> hotel capabilities (got string, but no caps parsed)")
			continue
		}

		for i, cp := range capabilities {
			if i == 0 {
				table.AddRow(pr, cp)
			} else {
				table.AddRow("", cp)
			}
		}
		table.AddRow("", "")
	}
	log.Printf("\n%s\n", table.Render())
}

func capList(capsString string) []string {
	list := []string{}
	for _, capability := range strings.Split(capsString, ",") {
		cp := strings.TrimLeft(strings.TrimSpace(capability), "engine.Cap")
		cp = strings.TrimLeft(cp, "Av")
		list = append(list, cp)
	}
	return list
}

func readCapabilityString(cstr string, avStart int, provider string) string {
	insideCaps := false

	capsString := ""
	for i := avStart; i < len(cstr); i++ {
		ch := string(cstr[i])
		if ch == "(" {
			insideCaps = true
			continue
		}
		if !insideCaps {
			continue
		}
		if ch == ")" {
			if insideCaps {
				break
			} else {
				log.Println("[WARN] unexpected format:", provider)
				capsString = ""
				break
			}
		}
		capsString += ch
	}
	return capsString
}

func usage() {
	log.Println("usage if $GOPATH is set: tpp-caps")
	log.Println("usage if you want manually locate tpp-providers: tpp-caps <tpp-providers path>")
}
