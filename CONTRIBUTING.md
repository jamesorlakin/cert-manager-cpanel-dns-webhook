# Contributing

Contributions of almost any kind are welcome!

## Building multi-arch

If intending to build your own edition the basic Dockerfile is simple enough.
If you'd like to be a multi-architecture image then a few extra commands are required.
Docker's Buildx plugin can provide multiple architecture builds via static emulation and automatic pushing of manifests:

```bash
# If you don't have `docker buildx`, install the buildx plugin
wget https://github.com/docker/buildx/releases/download/v0.9.1/buildx-v0.9.1.linux-amd64
mkdir -p ~/.docker/cli-plugins
mv buildx-v0.9.1.linux-amd64 ~/.docker/cli-plugins/docker-buildx
chmod +x ~/.docker/cli-plugins/docker-buildx

# Prepare cross-arch executing on the local machine
docker run --privileged --rm tonistiigi/binfmt --install all
docker buildx create --use

# Build and push
docker buildx build --push --platform linux/amd64,linux/arm64 -t whatever-registry:tag .
```

## Exporting the Helm chart into manifests

Creating the manifests for those using `kubectl` for installation can be performed via:
```bash
cd deploy/chart
helm template cpanel-webhook . > ../v0.1.0.yaml
```

The version number in `Chart.yaml` as well as the referenced image in `values.yaml` should be changed for every new release.
It's also probably worth remembering to tweak the linked yaml version within the README as well.
