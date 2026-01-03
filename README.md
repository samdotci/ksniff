# ksniff

[![Build Status](https://travis-ci.org/eldadru/ksniff.svg?branch=master)](https://travis-ci.org/eldadru/ksniff)

A kubectl plugin that utilize tcpdump and Wireshark to start a remote capture on any pod in your
 Kubernetes cluster.

You get the full power of Wireshark with minimal impact on your running pods.

### Intro

When working with micro-services, many times it's very helpful to get a capture of the network
activity between your micro-service and it's dependencies.

ksniff uses Kubernetes ephemeral containers to attach a tcpdump container to your target pod,
redirecting its output to your local Wireshark for smooth network debugging experience.

### Demo
![Demo!](https://i.imgur.com/hWtF9r2.gif)

### Requirements

- **Kubernetes 1.23+** (ephemeral containers became stable in 1.23)
- Wireshark 3.4.0+ (if using the GUI)

### Production Readiness
Ksniff [isn't production ready yet](https://github.com/eldadru/ksniff/issues/96#issuecomment-762454991), running ksniff for production workloads isn't recommended at this point.

## Installation
Installation via krew (https://github.com/GoogleContainerTools/krew)

    kubectl krew install sniff
    
For manual installation, download the latest release package, unzip it and use the attached makefile:  

    unzip ksniff.zip
    make install

### Wireshark

If you are using Wireshark with ksniff you must use at least version 3.4.0. Using older versions may result in issues reading captures (see [Known Issues](#known-issues) below).

## Build

Requirements:
1. go 1.11 or newer

Compiling:
 
    linux:      make linux
    windows:    make windows
    mac:        make darwin
 
### Usage

    kubectl sniff <POD_NAME> [-n <NAMESPACE_NAME>] [-c <CONTAINER_NAME>] [-i <INTERFACE_NAME>] [-f <CAPTURE_FILTER>] [-o OUTPUT_FILE]
    
    POD_NAME: Required. the name of the kubernetes pod to start capture it's traffic.
    NAMESPACE_NAME: Optional. Namespace name. used to specify the target namespace to operate on.
    CONTAINER_NAME: Optional. If omitted, the first container in the pod will be chosen.
    INTERFACE_NAME: Optional. Pod Interface to capture from. If omitted, all Pod interfaces will be captured.
    CAPTURE_FILTER: Optional. specify a specific tcpdump capture filter. If omitted no filter will be used.
    OUTPUT_FILE: Optional. if specified, ksniff will redirect tcpdump output to local file instead of wireshark. Use '-' for stdout.

#### Additional Options

    -t, --timeout: Timeout for ephemeral container to become ready (default: 1m)
    --tcpdump-image: Custom container image with tcpdump (default: nicolaka/netshoot:v0.14)
    -x, --context: kubectl context to work on
    -v, --verbose: Enable debug output

#### Air gapped environments
Use `--tcpdump-image` flag (or KUBECTL_PLUGINS_LOCAL_FLAG_TCPDUMP_IMAGE environment variable) to override the default container image:
  
    kubectl sniff <POD_NAME> [-n <NAMESPACE_NAME>] --tcpdump-image <PRIVATE_REPO>/netshoot

#### How it Works

ksniff creates an ephemeral container in the target pod that shares the network namespace with your target container. This ephemeral container runs tcpdump and streams the capture data back to your local machine.

Benefits of ephemeral containers:

- No file uploads required
- Works with any container (including scratch containers)
- No privileged pods needed
- Clean approach using native Kubernetes features
- Automatic cleanup when the pod is deleted

#### Piping output to stdout
By default ksniff will attempt to start a local instance of the Wireshark GUI. You can integrate with other tools
using the `-o -` flag to pipe packet cap data to stdout.

Example using `tshark`:

    kubectl sniff pod-name -f "port 80" -o - | tshark -r -

### Contribution
More than welcome! please don't hesitate to open bugs, questions, pull requests 

### Known Issues

#### Wireshark and TShark cannot read pcap

*Issues [100](https://github.com/eldadru/ksniff/issues/100) and [98](https://github.com/eldadru/ksniff/issues/98)*

Wireshark may show `UNKNOWN` in Protocol column. TShark may report the following in output:

```
tshark: The standard input contains record data that TShark doesn't support.
(pcap: network type 276 unknown or unsupported)
```

This issue happens when using an old version of Wireshark or TShark to read the pcap created by ksniff. Upgrade Wireshark or TShark to resolve this issue. Ubuntu LTS versions may have this problem with stock package versions but using the [Wireshark PPA will help](https://github.com/eldadru/ksniff/issues/100#issuecomment-789503442).
