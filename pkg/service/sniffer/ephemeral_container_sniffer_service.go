package sniffer

import (
	"io"
	"ksniff/kube"
	"ksniff/pkg/config"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	EphemeralContainerName = "ksniff-debug"
	DefaultTcpdumpImage    = "docker.io/nicolaka/netshoot:v0.14"
)

type EphemeralContainerSnifferService struct {
	settings             *config.KsniffSettings
	kubernetesApiService kube.KubernetesApiService
}

func NewEphemeralContainerSnifferService(options *config.KsniffSettings, service kube.KubernetesApiService) SnifferService {
	return &EphemeralContainerSnifferService{settings: options, kubernetesApiService: service}
}

func (e *EphemeralContainerSnifferService) Setup() error {
	log.Infof("creating ephemeral container in pod: '%s'", e.settings.UserSpecifiedPodName)

	tcpdumpImage := e.settings.TCPDumpImage
	if tcpdumpImage == "" {
		tcpdumpImage = DefaultTcpdumpImage
	}

	err := e.kubernetesApiService.CreateEphemeralContainer(
		e.settings.UserSpecifiedPodName,
		e.settings.UserSpecifiedContainer,
		EphemeralContainerName,
		tcpdumpImage,
		e.settings.UserSpecifiedContainerTimeout,
	)

	if err != nil {
		log.WithError(err).Errorf("failed to create ephemeral container in pod: '%s'", e.settings.UserSpecifiedPodName)
		return err
	}

	log.Info("ephemeral container created successfully")

	return nil
}

func (e *EphemeralContainerSnifferService) Cleanup() error {
	// Ephemeral containers cannot be removed - they exist until the pod is deleted.
	// This is by Kubernetes design, so cleanup is a no-op.
	log.Debug("ephemeral container cleanup not needed (auto-cleanup with pod)")
	return nil
}

func (e *EphemeralContainerSnifferService) Start(stdOut io.Writer) error {
	log.Info("starting remote sniffing using ephemeral container")

	command := []string{"tcpdump", "-i", e.settings.UserSpecifiedInterface,
		"-U", "-w", "-", e.settings.UserSpecifiedFilter}

	exitCode, err := e.kubernetesApiService.ExecuteCommand(
		e.settings.UserSpecifiedPodName,
		EphemeralContainerName,
		command,
		stdOut,
	)
	if err != nil || exitCode != 0 {
		if err != nil {
			return errors.Wrapf(err, "tcpdump failed, exit code: '%d'", exitCode)
		}
		return errors.Errorf("tcpdump failed, exit code: '%d'", exitCode)
	}

	log.Info("remote sniffing using ephemeral container completed")

	return nil
}
