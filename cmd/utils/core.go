package utils

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)


type Context struct {
	Name, AbsPath string
}



// func InitialProject(c *cli.Context) (*Context, error) {
// 	return Initial(c)
// }

// get server name
func Initial(c *cli.Context) (*Context, error) {
	absPath, err := filepath.Abs("./")
	if err != nil {
		return nil, errors.New("get project absolute path failure, reason: " + err.Error())
	}
	absPath = strings.Replace(absPath, `\`, `/`, -1)
	absPath = strings.TrimRight(absPath, "/")
	var (
		ctx = &Context{
			Name:    filepath.Base(absPath),
			AbsPath: absPath,
		}
	)
	return ctx, nil
}
