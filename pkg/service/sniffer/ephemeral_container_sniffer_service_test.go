package sniffer

import (
	"bytes"
	"errors"
	"io"
	"ksniff/pkg/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// MockKubernetesApiService is a mock implementation of KubernetesApiService for testing
type MockKubernetesApiService struct {
	CreateEphemeralContainerFunc func(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error
	ExecuteCommandFunc           func(podName string, containerName string, command []string, stdOut io.Writer) (int, error)
}

func (m *MockKubernetesApiService) CreateEphemeralContainer(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error {
	if m.CreateEphemeralContainerFunc != nil {
		return m.CreateEphemeralContainerFunc(podName, targetContainer, ephemeralContainerName, tcpdumpImage, timeout)
	}
	return nil
}

func (m *MockKubernetesApiService) ExecuteCommand(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {
	if m.ExecuteCommandFunc != nil {
		return m.ExecuteCommandFunc(podName, containerName, command, stdOut)
	}
	return 0, nil
}

// TestNewEphemeralContainerSnifferService tests the constructor
func TestNewEphemeralContainerSnifferService(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	mockService := &MockKubernetesApiService{}

	// when
	service := NewEphemeralContainerSnifferService(settings, mockService)

	// then
	assert.NotNil(t, service)
	_, ok := service.(*EphemeralContainerSnifferService)
	assert.True(t, ok)
}

// TestSetup_DefaultTcpdumpImage tests Setup with default tcpdump image
func TestSetup_DefaultTcpdumpImage(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedContainer = "test-container"
	settings.UserSpecifiedContainerTimeout = 30 * time.Second

	var capturedPodName, capturedTargetContainer, capturedEphemeralName, capturedImage string
	var capturedTimeout time.Duration

	mockService := &MockKubernetesApiService{
		CreateEphemeralContainerFunc: func(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error {
			capturedPodName = podName
			capturedTargetContainer = targetContainer
			capturedEphemeralName = ephemeralContainerName
			capturedImage = tcpdumpImage
			capturedTimeout = timeout
			return nil
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)

	// when
	err := service.Setup()

	// then
	assert.Nil(t, err)
	assert.Equal(t, "test-pod", capturedPodName)
	assert.Equal(t, "test-container", capturedTargetContainer)
	assert.Equal(t, EphemeralContainerName, capturedEphemeralName)
	assert.Equal(t, DefaultTcpdumpImage, capturedImage)
	assert.Equal(t, 30*time.Second, capturedTimeout)
}

// TestSetup_CustomTcpdumpImage tests Setup with custom tcpdump image
func TestSetup_CustomTcpdumpImage(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedContainer = "test-container"
	settings.TCPDumpImage = "custom/tcpdump:latest"
	settings.UserSpecifiedContainerTimeout = 30 * time.Second

	var capturedImage string

	mockService := &MockKubernetesApiService{
		CreateEphemeralContainerFunc: func(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error {
			capturedImage = tcpdumpImage
			return nil
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)

	// when
	err := service.Setup()

	// then
	assert.Nil(t, err)
	assert.Equal(t, "custom/tcpdump:latest", capturedImage)
}

// TestSetup_Error tests Setup when CreateEphemeralContainer fails
func TestSetup_Error(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedContainer = "test-container"

	mockService := &MockKubernetesApiService{
		CreateEphemeralContainerFunc: func(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error {
			return errors.New("failed to create ephemeral container")
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)

	// when
	err := service.Setup()

	// then
	assert.NotNil(t, err)
	assert.Equal(t, "failed to create ephemeral container", err.Error())
}

// TestCleanup tests Cleanup method (no-op)
func TestCleanup(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	mockService := &MockKubernetesApiService{}
	service := NewEphemeralContainerSnifferService(settings, mockService)

	// when
	err := service.Cleanup()

	// then
	assert.Nil(t, err)
}

// TestStart_Success tests Start with successful execution
func TestStart_Success(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedInterface = "eth0"
	settings.UserSpecifiedFilter = "tcp port 80"

	var capturedPodName, capturedContainerName string
	var capturedCommand []string
	var capturedStdOut io.Writer

	mockService := &MockKubernetesApiService{
		ExecuteCommandFunc: func(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {
			capturedPodName = podName
			capturedContainerName = containerName
			capturedCommand = command
			capturedStdOut = stdOut
			// Write some test data to stdOut to verify it's passed through
			stdOut.Write([]byte("test output"))
			return 0, nil
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)
	output := &bytes.Buffer{}

	// when
	err := service.Start(output)

	// then
	assert.Nil(t, err)
	assert.Equal(t, "test-pod", capturedPodName)
	assert.Equal(t, EphemeralContainerName, capturedContainerName)
	assert.Equal(t, []string{"tcpdump", "-i", "eth0", "-U", "-w", "-", "tcp port 80"}, capturedCommand)
	assert.Equal(t, output, capturedStdOut, "stdOut should be passed through to ExecuteCommand")
	assert.Equal(t, "test output", output.String(), "output written by ExecuteCommand should appear in stdOut")
}

// TestStart_NonZeroExitCode tests Start when command exits with non-zero code
func TestStart_NonZeroExitCode(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedInterface = "eth0"
	settings.UserSpecifiedFilter = ""

	mockService := &MockKubernetesApiService{
		ExecuteCommandFunc: func(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {
			return 1, nil
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)
	output := &bytes.Buffer{}

	// when
	err := service.Start(output)

	// then
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "tcpdump failed")
	assert.Contains(t, err.Error(), "exit code: '1'")
}

// TestStart_ExecutionError tests Start when ExecuteCommand returns an error
func TestStart_ExecutionError(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedInterface = "eth0"
	settings.UserSpecifiedFilter = ""

	mockService := &MockKubernetesApiService{
		ExecuteCommandFunc: func(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {
			return 0, errors.New("connection failed")
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)
	output := &bytes.Buffer{}

	// when
	err := service.Start(output)

	// then
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "tcpdump failed")
	assert.Contains(t, err.Error(), "exit code: '0'")
	assert.Contains(t, err.Error(), "connection failed", "original error message should be preserved")
}

// TestStart_BothErrorAndExitCode tests Start when both error and non-zero exit code occur
func TestStart_BothErrorAndExitCode(t *testing.T) {
	// given
	settings := config.NewKsniffSettings(genericclioptions.IOStreams{})
	settings.UserSpecifiedPodName = "test-pod"
	settings.UserSpecifiedInterface = "eth0"
	settings.UserSpecifiedFilter = ""

	mockService := &MockKubernetesApiService{
		ExecuteCommandFunc: func(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {
			return 127, errors.New("command not found")
		},
	}

	service := NewEphemeralContainerSnifferService(settings, mockService)
	output := &bytes.Buffer{}

	// when
	err := service.Start(output)

	// then
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "tcpdump failed")
	assert.Contains(t, err.Error(), "exit code: '127'")
	assert.Contains(t, err.Error(), "command not found", "original error message should be preserved")
}
