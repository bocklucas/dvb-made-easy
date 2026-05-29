package restore

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bocklucas/dvb-made-easy/internal/docker"
)

var safeFilenamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func BuildExtractionConfig(filename, stagingVolume, stagingSubdir, targetVolume, passphrase string) (docker.OneShotConfig, error) {
	if !safeFilenamePattern.MatchString(filename) {
		return docker.OneShotConfig{}, fmt.Errorf("unsafe backup filename: %q", filename)
	}

	isEncrypted := strings.HasSuffix(filename, ".gpg")

	env := []string{"BACKUP_FILE=/staging/" + stagingSubdir + "/" + filename}
	var cmd string

	if isEncrypted {
		cmd = `apk add --no-cache gnupg && gpg --batch --passphrase "$GPG_PASSPHRASE" -d "$BACKUP_FILE" | tar -xz -C /target --strip-components 2`
		env = append(env, "GPG_PASSPHRASE="+passphrase)
	} else {
		cmd = `tar -xzf "$BACKUP_FILE" -C /target --strip-components 2`
	}

	return docker.OneShotConfig{
		Image: "alpine:latest",
		Cmd:   []string{"sh", "-c", cmd},
		Env:   env,
		Mounts: []docker.Mount{
			{Source: stagingVolume, Target: "/staging", ReadOnly: true},
			{Source: targetVolume, Target: "/target", ReadOnly: false},
		},
	}, nil
}
