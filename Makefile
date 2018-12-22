# Setup name variables for the package/tool
NAME := gcp-cloud-compute-operator
PKG := github.com/paulczar/$(NAME)

CGO_ENABLED := 0

# Set any default go build tags.
BUILDTAGS :=

include basic.mk
