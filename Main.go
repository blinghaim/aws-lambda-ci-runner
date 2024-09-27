package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	gitv5 "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	sshgit "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ScriptRequest struct {
	SourceRepo            string `json:"source_repo"`
	Tag                   string `json:"tag"`
	MicrserviceName       string `json:"micrservice_name"`
	CiType                string `json:"ci_type"`
	InstallOnlyModuleName string `json:"install_only_module_name"`
}
type ScriptResponse struct {
	Output       string `json:"output"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var scriptRequest ScriptRequest
	err := json.Unmarshal([]byte(request.Body), &scriptRequest)
	if len(scriptRequest.MicrserviceName) == 0 {
		return ErrorMessage("micrservice_name is not set.")
	}
	if len(scriptRequest.Tag) == 0 {
		return ErrorMessage("tag is not set.")
	}
	if len(scriptRequest.SourceRepo) == 0 {
		return ErrorMessage("source_repo is not set.")
	}
	if len(scriptRequest.CiType) == 0 {
		return ErrorMessage("ci_type is not set.")
	}

	if err != nil {
		return ErrorMessage("Invalid Input:" + err.Error())
	}
	githubToken := os.Getenv("GITHUB_PRIVATE_KEY")
	if githubToken == "" {
		return ErrorMessage("GITHUB_PRIVATE_KEY environment variable is not set.")
	}

	directory := "/tmp/source"

	repoURL := scriptRequest.SourceRepo
	fmt.Println("Cloning private repository...:" + repoURL)
	if _, err := os.Stat(directory); err == nil {
		os.RemoveAll(directory)
	}
	referName := plumbing.NewTagReferenceName(scriptRequest.Tag)
	if scriptRequest.Tag == "dev" {
		referName = plumbing.NewBranchReferenceName(scriptRequest.Tag)
	}
	signer, err := ssh.ParsePrivateKey([]byte(githubToken))

	_, err = gitv5.PlainClone(directory, false, &gitv5.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		Auth: &sshgit.PublicKeys{
			User:   "git",
			Signer: signer,
			HostKeyCallbackHelper: sshgit.HostKeyCallbackHelper{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
		},
		ReferenceName: referName,
		SingleBranch:  true,
		Depth:         1,
	})

	if err != nil {
		return ErrorMessage("Error cloning repository:" + err.Error())
	}
	fmt.Println("Private repository cloned successfully.")

	if len(scriptRequest.InstallOnlyModuleName) > 0 {
		if output, err := runCommand(directory, "mkdir", "-p", "/mnt/home/go/pkg/mod/"+scriptRequest.InstallOnlyModuleName+"@"+scriptRequest.Tag); err != nil {
			return ErrorMessage("Error running 'mkdir -p  /mnt/home/go/pkg/mod/" + scriptRequest.InstallOnlyModuleName + "@" + scriptRequest.Tag + "': " + output)
		}
		if output, err := runCommand(directory, "cp", "-rf", "*", "/mnt/home/go/pkg/mod/"+scriptRequest.InstallOnlyModuleName+"@"+scriptRequest.Tag+"/"); err != nil {
			return ErrorMessage("Error running 'cp -rf * /mnt/home/go/pkg/mod/" + scriptRequest.InstallOnlyModuleName + "@" + scriptRequest.Tag + "/" + "': " + output)
		}
		return SuccessMessage("Installed:" + "/mnt/home/go/pkg/mod/" + scriptRequest.InstallOnlyModuleName + "@" + scriptRequest.Tag)
	}

	if scriptRequest.CiType == "go" {
		if output, err := runCommand(directory, "go", "mod", "tidy"); err != nil {
			return ErrorMessage("Error running 'go mod tidy': " + output)
		}
		buildCmd := []string{"go", "build", "-tags=nomsgpack", "-o", "./app"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}
		fmt.Println("Build completed successfully.")

		buildCmd = []string{"mkdir", "-p", "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		buildCmd = []string{"cp", "-rf", "./app", "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}
		fmt.Println("Build completed successfully.")
		return SuccessMessage(scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/app")
	} else if scriptRequest.CiType == "npm" {

		buildCmd := []string{"mkdir", "-p", "/mnt/home/" + scriptRequest.MicrserviceName + "/node_modules"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		buildCmd = []string{"sh", "-c", "npm config set registry http://registry.npmjs.org"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		buildCmd = []string{"sh", "-c", "npm install --maxsockets=1"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		cmds := []string{"sh", "-c", "ls -rthl"}
		outputUlimit, _ := runCommand(directory, cmds[0], cmds[1:]...)
		fmt.Println("ls -rthl\n", outputUlimit)

		buildCmd = []string{"sh", "-c", "npm run build"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		buildCmd = []string{"mkdir", "-p", "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		cmd := "cp -rf build " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"
		buildCmd = strings.Split(cmd, " ")
		if _, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			cmd = "cp -rf out " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"
			buildCmd = strings.Split(cmd, " ")
			if _, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
				cmd = "cp -rf dist " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"
				buildCmd = strings.Split(cmd, " ")
				if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
					return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
				}
			}
		}

		return SuccessMessage(scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist")
	} else if scriptRequest.CiType == "yarn" {

		buildCmd := []string{"mkdir", "-p", "/mnt/home/" + scriptRequest.MicrserviceName + "/node_modules"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		cmd := "npm config set registry http://registry.npmjs.org"
		buildCmd = strings.Split(cmd, " ")
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + cmd + "': " + output)
		}

		cmd = "yarn install"
		buildCmd = strings.Split(cmd, " ")
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + cmd + "': " + output)
		}

		cmd = "yarn build"
		buildCmd = strings.Split(cmd, " ")
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + cmd + "': " + output)
		}

		buildCmd = []string{"mkdir", "-p", "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}

		cmd = "cp -rf build " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"
		buildCmd = strings.Split(cmd, " ")
		if _, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			cmd = "cp -rf out " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"
			buildCmd = strings.Split(cmd, " ")
			if _, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
				cmd = "cp -rf dist " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"
				buildCmd = strings.Split(cmd, " ")
				if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
					return ErrorMessage("Error running '" + cmd + "': " + output)
				}
			}
		}

		return SuccessMessage(scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist")
	} else if scriptRequest.CiType == "nodejs" {
		buildCmd := []string{"mkdir", "-p", "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist"}
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}
		cmd := "cp -rfa * " + "/mnt/home/" + scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist/"
		buildCmd = strings.Split(cmd, " ")
		if output, err := runCommand(directory, buildCmd[0], buildCmd[1:]...); err != nil {
			return ErrorMessage("Error running '" + strings.Join(buildCmd, " ") + "': " + output)
		}
		return SuccessMessage(scriptRequest.MicrserviceName + "/" + scriptRequest.Tag + "/dist")
	} else {
		return ErrorMessage("Error Citype not handled: " + scriptRequest.CiType)
	}
}

func ErrorMessage(errmsg string) (events.APIGatewayProxyResponse, error) {
	response := ScriptResponse{}
	response.ErrorMessage = errmsg
	response.Output = errmsg
	responseBody, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(responseBody)}, nil
}
func SuccessMessage(msg string) (events.APIGatewayProxyResponse, error) {
	response := ScriptResponse{}
	response.Output = msg
	responseBody, _ := json.Marshal(response)
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: string(responseBody)}, nil
}

func main() {
	lambda.Start(Handler)
}

func runCommand(directory string, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Env = os.Environ()

	cmd.Dir = directory

	var outBuf, errBuf bytes.Buffer

	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)
	fmt.Println("Command Running:" + command + " " + strings.Join(args, " "))
	beforetime := time.Now().Unix()
	err := cmd.Run()
	afterTime := time.Now().Unix()
	output := outBuf.String() + errBuf.String()

	if err != nil {
		fmt.Printf("Command finished with error: %v\n", err)
		return output, err
	}

	fmt.Println("Command finished successfully ( Duration: " + fmt.Sprint((afterTime - beforetime)) + " seconds ) " + command + " " + strings.Join(args, " "))
	return output, nil
}
