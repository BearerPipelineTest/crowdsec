//go:build linux || freebsd || netbsd || openbsd || solaris || !windows

package csplugin

import (
	"fmt"
	"io/fs"
	"math"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

func CheckOwner(details fs.FileInfo, path string) error {
	// check if it is owned by current user
	currentUser, err := user.Current()
	if err != nil {
		return errors.Wrap(err, "while getting current user")
	}
	currentUID, err := getUID(currentUser.Username)
	if err != nil {
		return errors.Wrap(err, "while looking up the current uid")
	}
	stat := details.Sys().(*syscall.Stat_t)
	if stat.Uid != currentUID {
		return fmt.Errorf("plugin at %s is not owned by user '%s'", path, currentUser.Username)
	}
	return nil
}

func CheckCredential(uid int, gid int) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}
}

func (pb *PluginBroker) CreateCmd(binaryPath string) (*exec.Cmd, error) {
	var err error
	cmd := exec.Command(binaryPath)
	cmd.SysProcAttr, err = getProcessAttr(pb.pluginProcConfig.User, pb.pluginProcConfig.Group)
	if err != nil {
		return nil, errors.Wrap(err, "while getting process attributes")
	}
	cmd.SysProcAttr.Credential.NoSetGroups = true
	return cmd, err
}

func getPluginTypeAndSubtypeFromPath(path string) (string, string, error) {
	pluginFileName := filepath.Base(path)
	parts := strings.Split(pluginFileName, "-")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("plugin name %s is invalid. Name should be like {type-name}", path)
	}
	return strings.Join(parts[:len(parts)-1], "-"), parts[len(parts)-1], nil
}

func getUID(username string) (uint32, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, err
	}
	uid, err := strconv.ParseInt(u.Uid, 10, 32)
	if err != nil {
		return 0, err
	}
	if uid < 0 || uid > math.MaxInt32 {
		return 0, fmt.Errorf("out of bound uid")
	}
	return uint32(uid), nil
}

func getGID(groupname string) (uint32, error) {
	g, err := user.LookupGroup(groupname)
	if err != nil {
		return 0, err
	}
	gid, err := strconv.ParseInt(g.Gid, 10, 32)
	if err != nil {
		return 0, err
	}
	if gid < 0 || gid > math.MaxInt32 {
		return 0, fmt.Errorf("out of bound gid")
	}
	return uint32(gid), nil
}

func getProcessAttr(username string, groupname string) (*syscall.SysProcAttr, error) {
	uid, err := getUID(username)
	if err != nil {
		return nil, err
	}
	gid, err := getGID(groupname)
	if err != nil {
		return nil, err
	}
	return &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uid,
			Gid: gid,
		},
	}, nil
}
