package jenkins

import (
	"a/kit/cmd/config"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bndr/gojenkins"
	"github.com/urfave/cli/v2"
)

// 读取配置文件
var (
	appName  string
	appGroup string
	Flags    = []cli.Flag{
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "Set jenkins appName",
		},
		&cli.StringFlag{
			Name:    "group",
			Aliases: []string{"g"},
			Usage:   "Set jenkins appGroup",
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

// generate job configuration file
func genConfigString(name, group string) string {
	str := strings.Replace(config.JenkinsJobConfig, "${app_name}", name, -1)
	str = strings.Replace(str, "${app_group}", group, -1)
	return str
}
func RunCreateJob(c *cli.Context) error {
	return createJob(c)
}
func createJob(c *cli.Context) error {
	ctx := context.Background()
	jenkins := gojenkins.CreateJenkins(nil, cfg.Jenkins.Url, cfg.Jenkins.Username, cfg.Jenkins.Password)
	_, err := jenkins.Init(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("开始新建任务%s\n", appName)
	configStr := genConfigString(appName, appGroup)
	_, err = jenkins.CreateJob(ctx, configStr, appName)
	if err != nil {
		fmt.Printf("err:%v\n", err)
		return err
	}
	fmt.Printf("%s创建成功\n", appName)
	return nil

}
