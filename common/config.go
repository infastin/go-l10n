package common

import (
	"regexp"

	"github.com/alecthomas/kong"
	"github.com/infastin/go-l10n/ast"
)

const cliVersion = "v1.0.6"

var Config struct {
	Directory         string
	PackageName       string
	Output            string
	Pattern           regexp.Regexp
	FormatSpecifiers  []rune
	SpecifierToGoType map[rune]ast.GoType
	Imports           []ast.GoImport
}

var cli struct {
	Dir     string           `required:"" short:"d" type:"existingdir" placeholder:"DIR" help:"Path to the directory with localization files."`
	Pattern string           `optional:"" short:"p" default:"${pattern}" placeholder:"PATTERN" help:"Localization file regexp pattern."`
	Package string           `optional:"" short:"P" default:"${package}" help:"Package name."`
	Output  string           `required:"" short:"o" placeholder:"DIR" help:"Path to output directory."`
	Version kong.VersionFlag `optional:"" short:"v" help:"Print version number."`
}

func InitConfig() {
	kong.Parse(&cli,
		kong.Description("Simple command-line utility to localize your Golang applications."),
		kong.Vars{
			"pattern": `([a-z_]+)\.([a-z_]+)\.(yaml|yml|json|toml)`,
			"package": "l10n",
			"version": cliVersion,
		},
	)

	Config.Directory = cli.Dir
	Config.Pattern = *regexp.MustCompile(cli.Pattern)
	Config.PackageName = cli.Package
	Config.Output = cli.Output

	Config.FormatSpecifiers = []rune{'s', 'd', 'f', 'S', 'F', 'M'}
	Config.SpecifierToGoType = map[rune]ast.GoType{
		's': {Type: "string"},
		'd': {Type: "int"},
		'f': {Type: "float64"},
		'v': {Type: "any"},
		'S': {
			Import:  "fmt",
			Package: "fmt",
			Type:    "Stringer",
		},
	}
}
