package config

import (
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type KsniffSettings struct {
	UserSpecifiedPodName          string
	UserSpecifiedInterface        string
	UserSpecifiedFilter           string
	UserSpecifiedContainer        string
	UserSpecifiedNamespace        string
	UserSpecifiedOutputFile       string
	UserSpecifiedVerboseMode      bool
	UserSpecifiedKubeContext      string
	UserSpecifiedContainerTimeout time.Duration
	TCPDumpImage                  string
}

func NewKsniffSettings(streams genericclioptions.IOStreams) *KsniffSettings {
	return &KsniffSettings{}
}
