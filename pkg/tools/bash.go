package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"chewbacca/internal/util"
)

type BashInput struct {
	Command     string `json:"command" jsonschema_description:"Bash command to run"`
	Description string `json:"description" jsonschema_description:"Why I'm running this command"`
	Timeout     int    `json:"timeout,omitempty" jsonschema_description:"Timeout in seconds (default: 30)"`
	WorkDir     string `json:"workdir,omitempty" jsonschema_description:"Working directory for the command"`
}

type BashResult struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ReturnCode int    `json:"return_code"`
	TimeTaken  string `json:"time_taken"`
}

func Bash(input json.RawMessage) (string, error) {
	var bashInput BashInput
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if bashInput.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := util.GetTimeoutOrDefault(bashInput.Timeout, 30)

	result, err := executeCommand(bashInput.Command, bashInput.WorkDir, timeout)
	if err != nil {
		return "", err
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

func executeCommand(command, workDir string, timeout int) (*BashResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	result := &BashResult{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		TimeTaken: elapsed.Round(time.Millisecond).String(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.ReturnCode = -1
		return result, fmt.Errorf("command timed out after %d seconds", timeout)
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ReturnCode = exitError.ExitCode()
		} else {
			result.ReturnCode = -1
		}
		return result, nil
	}

	result.ReturnCode = 0
	return result, nil
}

var BashDefinition = ToolDefinition{
	Name: "Bash",
	Description: `Execute a bash command in the shell.
Use this to run system commands, scripts, or any CLI operations.
The command runs with a default timeout of 30 seconds.`,
	InputSchema: GenerateSchema[BashInput](),
	Function:    Bash,
}
