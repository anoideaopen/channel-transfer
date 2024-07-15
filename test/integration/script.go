package test

import (
	"errors"
	"os"
	"strings"
	"time"
)

const (
	getPIDscript = "#!/bin/sh\npgrep -f /app > /state/channel-transfer/pid & rm -f \"$CUSTOM_SCRIPT\""

	customScriptPath = "/script/.custom_script"
	idPath           = "/script/pid"
	statusPath       = "/script/status"
)

func getStatus(pid string) (string, error) {
	// sending get status to channel-transfer
	getStatusScript := []byte("#!/bin/sh\ncat /proc/" + pid + "/status | grep \"State\" > /state/channel-transfer/status & rm -f \"$CUSTOM_SCRIPT\"")
	err := os.WriteFile(customScriptPath, getStatusScript, 0644)
	if err != nil {
		return "", err
	}

	time.Sleep(time.Millisecond * 1500)

	file, err := os.ReadFile(statusPath)
	if err != nil {
		return "", err
	}

	err = os.Remove(statusPath)
	if err != nil {
		return "", err
	}
	return string(file), nil
}

func restartService() error {
	pid, err := getPID()
	if err != nil {
		return err
	}

	restartServiceScript := []byte("#!/bin/sh\nsleep 1 && kill " + pid + " & rm -f \"$CUSTOM_SCRIPT\"")
	err = os.WriteFile(customScriptPath, restartServiceScript, 0644)
	if err != nil {
		return err
	}

	return nil
}

func sigstopService() error {
	pid, err := getPID()
	if err != nil {
		return err
	}

	// sending sigstop to channel-transfer
	sigstopScript := []byte("#!/bin/sh\nkill -19 " + pid + " & rm -f \"$CUSTOM_SCRIPT\"")
	err = os.WriteFile(customScriptPath, sigstopScript, 0644)
	if err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 1500)

	err = waitStatusWithRetry(pid, "stopped")
	if err != nil {
		return err
	}

	return nil
}

func waitStatusWithRetry(pid string, status string) error {
	i := 0
	var s string
	for i < 30 {
		s, err := getStatus(pid)
		if err != nil {
			return err
		}
		if strings.Contains(s, status) {
			return nil
		} else {
			i++
			time.Sleep(time.Millisecond * 200)
		}
	}
	return errors.New("can't sigstop channel-transfer service, current service status is: " + s)
}

func sigcontService() error {
	pid, err := getPID()
	if err != nil {
		return err
	}

	// sending sigcont to channel-transfer
	sigcontScript := []byte("#!/bin/sh\nkill -18 " + pid + " & rm -f \"$CUSTOM_SCRIPT\"")
	err = os.WriteFile(customScriptPath, sigcontScript, 0644)
	if err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 1500)

	if err = waitStatusWithRetry(pid, "sleeping"); err != nil {
		return err
	}

	return nil
}

func getPID() (string, error) {
	// sending get pid script
	getPidScript := []byte(getPIDscript)
	err := os.WriteFile(customScriptPath, getPidScript, 0644)
	if err != nil {
		return "", err
	}

	time.Sleep(time.Millisecond * 1500)

	file, err := os.ReadFile(idPath)
	if err != nil {
		return "", err
	}

	pid := strings.TrimSuffix(string(file), "\n")

	err = os.Remove(idPath)
	if err != nil {
		return "", err
	}

	return pid, nil
}
