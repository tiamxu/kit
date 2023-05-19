package build

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	RegistryDomain    = "registry.cn-hangzhou.aliyuncs.com"
	RegistryNamespace = "unipal"
	Env               = "dev"
)

var (
	version      string
	env          string
	lang         string
	registryPath string
	Flags        = []cli.Flag{
		&cli.StringFlag{
			Name:    "tag",
			Aliases: []string{"t"},
			Value:   "latest",
			Usage:   "Set docker image tag",
		},
		&cli.StringFlag{
			Name:    "env",
			Aliases: []string{"e"},
			Value:   "dev",
			Usage:   "Set docker image env",
		},
		&cli.StringFlag{
			Name:    "lang",
			Aliases: []string{"l"},
			Value:   "go",
			Usage:   "Set type of code langue ",
		},
	}
)
var ctx *Context

func InitProject(c *cli.Context) (err error) {
	ctx, err = Initial(c)
	return
}
func InitFlags(c *cli.Context) error {
	version = c.String("tag")
	if version == "" {
		version = "latest"
	}
	env = c.String("env")
	if env == "" {
		env = Env
	}
	lang = c.String("lang")
	if lang == "" {
		lang = "go"
	}
	registryPath = fmt.Sprintf("%s/%s/%s:%s", RegistryDomain, RegistryNamespace, strings.ToLower(env+"_"+ctx.Name), version)
	return nil
}

func needSudo() bool {
	sysType := runtime.GOOS
	if sysType == "linux" {
		if os.Getuid() != 0 {
			return true
		}
	}
	return false
}
func RunBuild(c *cli.Context) error {
	if err := loginAction(c); err != nil {
		return err
	}
	return buildAction(c)
}
func RunPush(c *cli.Context) error {
	if err := loginAction(c); err != nil {
		return err
	}
	return pushAction(c)
}
func buildAction(c *cli.Context) error {
	//dockerfile
	if _, err := os.Stat("./Dockerfile-dev"); os.IsNotExist(err) {
		return errors.New("docker build failed,not found Dockerfile")
	}
	var (
		stderr bytes.Buffer
		stdout bytes.Buffer
	)
	//code langue build
	var codeBuilder *exec.Cmd
	fmt.Printf("lang:%s,env:%s\n", lang, env)
	if lang == "node" {
		cmd := "npm run build" + ":" + env
		exec.Command("npm", "install", "--registry=https://registry.npm.taobao.org")
		// codeBuilder = exec.Command("npm", "run", "build")
		codeBuilder = exec.Command("bash", "-c", cmd)

	} else {
		codeBuilder = exec.Command("go", "build", "-o", "bin/main")
		codeBuilder.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	}
	codeBuilder.Stderr = &stderr
	codeBuilder.Stdout = &stdout
	fmt.Println(codeBuilder.String())
	if err := codeBuilder.Run(); err != nil {
		fmt.Println(stderr.String())
		return err
	}
	fmt.Println(stdout.String())
	fmt.Printf("%s build complate\n", c.String("lang"))
	//docker image build
	var dockerBuilder *exec.Cmd
	if needSudo() {
		dockerBuilder = exec.Command("sudo", "docker", "build",
			"-f", "Dockerfile-dev",
			"-t", registryPath, ".")
	} else {
		dockerBuilder = exec.Command("docker", "build",
			"-f", "Dockerfile-dev",
			"-t", registryPath, ".")
	}
	dockerBuilder.Stderr = &stderr
	dockerBuilder.Stdout = &stdout
	fmt.Println(dockerBuilder.String())
	if err := dockerBuilder.Run(); err != nil {
		fmt.Println(stderr.String())
		return err
	}
	fmt.Println("docker build complate")
	return nil
}
func pushAction(c *cli.Context) error {
	var (
		pushCmd *exec.Cmd
		stderr  bytes.Buffer
		stdout  bytes.Buffer
	)
	if needSudo() {
		pushCmd = exec.Command("sudo", "docker", "push", registryPath)
	} else {
		pushCmd = exec.Command("docker", "push", registryPath)

	}
	pushCmd.Stderr = &stderr
	pushCmd.Stdout = &stdout
	fmt.Println(pushCmd.String())
	fmt.Println("docker push images")
	if err := pushCmd.Run(); err != nil {
		fmt.Println(stderr.String())
		return err
	}
	fmt.Println("docker push images complate")
	return nil
}
func loginAction(c *cli.Context) error {
	var (
		loginCmd *exec.Cmd
		stderr   bytes.Buffer
		stdout   bytes.Buffer
	)
	if needSudo() {
		loginCmd = exec.Command("sudo", "docker", "login",
			"--username=root", "--password=123456",
			"https://registry.cn-hangzhou.aliyuncs.com")
	} else {
		loginCmd = exec.Command("docker", "login", "--username=root", "--password=123456", "https://registry.cn-hangzhou.aliyuncs.com")
	}
	loginCmd.Stderr = &stderr
	loginCmd.Stdout = &stdout
	fmt.Println(loginCmd.String()) //打印执行命令
	// fmt.Println("docker login")
	if err := loginCmd.Run(); err != nil {
		fmt.Println(stderr.String())
		return fmt.Errorf("docker login fail, error: %s.stderr: %s", err, stderr.String())
	}
	fmt.Println("docker login success")

	return nil
}

//docker build --platform=linux/amd64 -t registry.cn-hangzhou.aliyuncs.com/unipal/test_admindashboard:latest .
//CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dpcd
