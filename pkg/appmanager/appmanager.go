package appmanager

import (
	"archive/tar"
	"bufio"
	"fmt"
	"github.com/fanux/sealos/install"
	"github.com/wonderivan/logger"
	"io"
	"os"
	"path"
	"strings"
	"sync"
)

//Command is
type Command struct {
	Name string // LOAD START APPLY DELETE STOP REMOVE
	Cmd  string // kubectl apply -k
	Type string
}

type PkgConfig struct {
	Cmds    []Command
	URL     string
	Name    string
	Workdir string
}

func nameFromUrl(url string) string {
	tmp := path.Base(url)
	name := strings.Split(tmp, ".tar")
	if len(name) < 1 {
		logger.Error("app package must *.tar, [%s] is invalid, %s", url, name)
		os.Exit(1)
	}
	return name[0]
}

func LoadAppConfig(url string, flagConfig string) (*PkgConfig, error) {
	var pkgConfig *PkgConfig
	var err error
	// 如果指定了config。 则直接从config里面读取配置
	if flagConfig == "" {
		pkgConfig, err = LoadConfig(url)
		if err != nil {
			logger.Error("load config failed: %s", err)
			os.Exit(0)
		}
	} else {
		f, err := os.Open(flagConfig)
		if err != nil {
			logger.Error("load config failed: %s", err)
			os.Exit(0)
		}
		pkgConfig, err = configFromReader(f)
	}
	return pkgConfig, nil
}

// LoadConfig from tar package
/*
kube.tar
   config
   images.tar

config content:

LOAD docker load -i images.tar
START systemctl start docker
DELETE docker rmi
STOP systemctl top
APPLY kubectl apply -f
*/
func LoadConfig(packageFile string) (*PkgConfig, error) {
	filename, _ := install.DownloadFile(packageFile)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(file)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == "config" {
			logger.Info("config content: ")
			config, err := configFromReader(tr)
			return config, err
		}
	}
	return nil, err
}

func configFromReader(reader io.Reader) (*PkgConfig, error) {
	config := &PkgConfig{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		command := Command{}
		text := scanner.Text()
		logger.Info(text) // Println will add back the final '\n'
		name, cmd, err := decodeCmd(text)
		if err != nil {
			logger.Error(err)
			continue
		}
		command.Name = name
		command.Cmd = cmd

		config.Cmds = append(config.Cmds, command)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config line failed %s", err)
	}
	return config, nil
}

// text is "LOAD docker load -i images.tar"
// name LOAD
// cmd docker load -i images.tar
func decodeCmd(text string) (name string, cmd string, err error) {
	list := strings.SplitN(text, " ", 2)
	if len(list) != 2 {
		return "", "", fmt.Errorf("Command fomat error:[%s]", text)
	}
	return list[0], list[1], nil
}

type Runner interface {
	Run(config install.SealConfig, pkgConfig *PkgConfig)
}

type RunOnEveryNodes struct {
	Cmd []Command
}

func (r *RunOnEveryNodes) Run(config install.SealConfig, p *PkgConfig) {
	var wg sync.WaitGroup
	workspace := fmt.Sprintf("%s/%s", p.Workdir, p.Name)
	nodes := append(config.Masters, config.Nodes...)
	// values.yaml 存在， 则将 values.yaml复制到各个节点。
	if Values == "-" {
		// 处理 stdin
		SendPackage(p.Workdir+"values.yaml", nodes, workspace, nil, nil)
	} else if Values != "" {
		SendPackage(Values, nodes, workspace, nil, nil)
	}
	// delete的时候只需要执行r.cmd里面的STOP/REMOVE命令即可
	if p.Flag == "install" {
		SendPackage(p.URL, nodes, workspace, nil, nil)
	}
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			// delete的时候只需要执行r.cmd里面的STOP/REMOVE命令即可
			if p.Flag == "install" {
				tarCmd := fmt.Sprintf("tar xvf %s.tar", p.Name)
				fmt.Println(tarCmd)
				CmdWorkSpace(node, tarCmd, workspace)
			}
			for _, cmd := range r.Cmd {
				CmdWorkSpace(node, cmd.Cmd, workspace)
			}
			// 删除 tar压缩包及解压缩目录
			if p.Flag == "delete" {
				// rm -rf  $workdir/$pkgName
				rmTar := fmt.Sprintf("rm -rf %s", workspace)
				CmdWorkSpace(node, rmTar, Workdir)
			}
		}(node)
	}

	wg.Wait()
}

type RunOnMaster struct {
	Cmd []Command
}

func (r *RunOnMaster) Run(config install.SealConfig, p *PkgConfig) {
	workspace := fmt.Sprintf("%s/%s", p.Workdir, p.Name)
	// delete的时候只需要执行r.cmd里面的DELETE命令即可
	if p.Flag == "install" {
		SendPackage(p.URL, []string{config.Masters[0]}, workspace, nil, nil)
		tarCmd := fmt.Sprintf("tar xvf %s.tar", p.Name)
		fmt.Println(tarCmd)
		CmdWorkSpace(config.Masters[0], tarCmd, workspace)
	}
	for _, cmd := range r.Cmd {
		CmdWorkSpace(config.Masters[0], cmd.Cmd, workspace)
	}
}

func CmdWorkSpace(node, cmd, workdir string) {
	command := fmt.Sprintf("cd %s && %s", workdir, cmd)
	_ = SSHConfig.CmdAsync(node, command)
}
