package gitlab

import (
	"errors"
	"fmt"
	"log"

	"github.com/tiamxu/kit/sql"

	"github.com/urfave/cli/v2"
	"github.com/xanzy/go-gitlab"
)

var (
	appName  string
	appGroup string
	Flags    = []cli.Flag{
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "set gitlab project name",
		},
		&cli.StringFlag{
			Name:    "group",
			Aliases: []string{"g"},
			Usage:   "set gitlab  group",
		},
	}
)

func init() {
	//load config
	loadConfig()
}
func InitFlags(c *cli.Context) error {
	appName = c.String("name")
	appGroup = c.String("group")
	if appName == "" || appGroup == "" {
		return errors.New("Missing required parameter")
	}
	return nil
}
func RunGetProject(c *cli.Context) error {
	return getProject(c)
}
func getProject(c *cli.Context) error {

	git, err := gitlab.NewClient(cfg.Gitlab.Token, gitlab.WithBaseURL(cfg.Gitlab.Url))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	// var gid int
	groupOption := &gitlab.ListGroupsOptions{Search: gitlab.String(appGroup)}
	groups, _, err := git.Groups.ListGroups(groupOption)
	if err != nil {
		log.Fatalf("Failed to get groups err: %v", err)
	}
	group := groups[0]
	fmt.Println(group.ID, group.Name)
	// gid = group.ID
	err = sql.Connect(cfg.DB)
	if err != nil {
		fmt.Printf("数据库连接错误,error:%s\n", err)
		return err
	}
	fmt.Println("数据库连接正常")
	return nil

}
