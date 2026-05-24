package restore

import (
	"strings"

	"github.com/offen/restore-manager/internal/docker"
)

func BuildExtractionConfig(filename, stagingVolume, stagingSubdir, targetVolume, passphrase string) docker.OneShotConfig {
	isEncrypted := strings.HasSuffix(filename, ".gpg")
	backupPath := "/staging/" + stagingSubdir + "/" + filename

	var cmd string
	var env []string

	if isEncrypted {
		cmd = `apk add --no-cache gnupg && gpg --batch --passphrase "$GPG_PASSPHRASE" -d ` + backupPath + ` | tar -xz -C /target --strip-components 2`
		env = []string{"GPG_PASSPHRASE=" + passphrase}
	} else {
		cmd = "tar -xzf " + backupPath + " -C /target --strip-components 2"
	}

	return docker.OneShotConfig{
		Image: "alpine:latest",
		Cmd:   []string{"sh", "-c", cmd},
		Env:   env,
		Mounts: []docker.Mount{
			{Source: stagingVolume, Target: "/staging", ReadOnly: true},
			{Source: targetVolume, Target: "/target", ReadOnly: false},
		},
	}
}
