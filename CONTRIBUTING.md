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
